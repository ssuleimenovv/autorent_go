package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"autorent-backend/pkg/messaging"
	"autorent-backend/pkg/observability"
	identityDB "autorent-backend/shared/identity-service/internal/database"
	identityModels "autorent-backend/shared/identity-service/internal/models"
	identityRepo "autorent-backend/shared/identity-service/internal/repository"
	identityService "autorent-backend/shared/identity-service/internal/service"
)
type IdentityService struct {
	db         *gorm.DB
	logger     *zap.Logger
	publisher  *messaging.RabbitMQPublisher
	userRepo   *identityRepo.UserRepository
	jwtManager *identityService.JWTManager
}
func main() {
	// Load environment variables
	godotenv.Load(".env")

	// Initialize logger
	logger, err := observability.InitLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize tracer
	ctx := context.Background()
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		otelEndpoint = "http://otel-collector:4318"
	}

	tp, err := observability.InitTracer(ctx, "identity-service", otelEndpoint)
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	defer tp.Shutdown(ctx)

	// Initialize database
	db, err := initDatabase()
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Initialize RabbitMQ
	conn, err := initRabbitMQ()
	if err != nil {
		logger.Fatal("Failed to initialize RabbitMQ", zap.Error(err))
	}
	defer conn.Close()

	publisher, err := messaging.NewRabbitMQPublisher(conn, "autorent.events")
	if err != nil {
		logger.Fatal("Failed to create RabbitMQ publisher", zap.Error(err))
	}

	if err := identityDB.Migrate(db); err != nil {
		logger.Fatal("Failed to migrate database", zap.Error(err))
	}
	if err := identityDB.SeedDefaultData(db); err != nil {
		logger.Warn("Failed to seed default data", zap.Error(err))
	}

	userRepo := identityRepo.NewUserRepository(db)
	privatePEM, publicPEM, err := generateDefaultJWTKeys()
	if err != nil {
		logger.Warn("Failed to generate JWT keys", zap.Error(err))
	}
	jwtManager, err := identityService.NewJWTManager(privatePEM, publicPEM)
	if err != nil {
		logger.Warn("Failed to initialize JWT manager, auth endpoints will be limited", zap.Error(err))
	}

	// Initialize Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Create service
	svc := &IdentityService{
		db:         db,
		logger:     logger,
		publisher:  publisher,
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}

	// Routes
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.GET("/metrics", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "metrics endpoint"})
	})

	e.POST("/auth/login", svc.Login)
	e.POST("/auth/refresh", svc.Refresh)
	e.POST("/auth/activate", svc.Activate)

	e.GET("/.well-known/jwks.json", svc.GetJWKS)

	e.GET("/users", svc.ListUsers)
	e.POST("/users", svc.CreateUser)
	e.GET("/users/:id", svc.GetUser)
	e.PATCH("/users/:id/activate", svc.ActivateUser)
	e.PATCH("/users/:id/deactivate", svc.DeactivateUser)
	e.DELETE("/users/:id", svc.DeleteUser)

	e.POST("/users/:id/roles/:roleId", svc.AssignRole)
	e.DELETE("/users/:id/roles/:roleId", svc.RemoveRole)

	e.GET("/roles", svc.ListRoles)
	e.POST("/roles", svc.CreateRole)
	e.POST("/roles/:id/permissions/:permissionId", svc.AssignPermission)
	e.DELETE("/roles/:id/permissions/:permissionId", svc.RemovePermission)

	e.GET("/permissions", svc.ListPermissions)
	e.POST("/permissions", svc.CreatePermission)

	// Internal endpoints
	e.POST("/internal/users/provision", svc.ProvisionUser)
	e.GET("/internal/users/:id", svc.GetUserInternal)

	// Start server
	go func() {
		logger.Info("Starting identity-service", zap.String("port", "8080"))
		if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.Error("Shutdown error", zap.Error(err))
	}

	logger.Info("Identity service stopped")
}

func initDatabase() (*gorm.DB, error) {
	dsn := os.Getenv("ConnectionStrings__DbConnection")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=postgres_db port=5432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

func initRabbitMQ() (*amqp.Connection, error) {
	host := os.Getenv("RabbitMq__HostName")
	if host == "" {
		host = "rabbitmq"
	}

	port := os.Getenv("RabbitMq__Port")
	if port == "" {
		port = "5672"
	}

	user := os.Getenv("RabbitMq__UserName")
	if user == "" {
		user = "autorent"
	}

	password := os.Getenv("RabbitMq__Password")
	if password == "" {
		password = "autorent"
	}

	dsn := fmt.Sprintf("amqp://%s:%s@%s:%s/", user, password, host, port)

	conn, err := amqp.Dial(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	return conn, nil
}

func generateDefaultJWTKeys() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})

	return privatePEM, publicPEM, nil
}

// Placeholder handlers
func (s *IdentityService) Login(c echo.Context) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if len(body) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "empty body"})
	}
	if err := json.Unmarshal(body, &req); err != nil {
		trimmed := string(body)
		trimmed = strings.TrimSpace(trimmed)
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			trimmed = strings.Trim(trimmed, "{}")
			parts := strings.Split(trimmed, ",")
			for _, part := range parts {
				kv := strings.SplitN(part, ":", 2)
				if len(kv) != 2 {
					continue
				}
				key := strings.TrimSpace(strings.Trim(kv[0], `"`))
				value := strings.TrimSpace(strings.Trim(kv[1], `"`))
				switch key {
				case "username":
					req.Username = value
				case "password":
					req.Password = value
				}
			}
		} else {
			s.logger.Error("login request payload parse failed", zap.Error(err), zap.String("body", string(body)))
			return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request", "detail": err.Error()})
		}
	}
	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "username and password are required"})
	}

	user, err := s.userRepo.GetByEmail(req.Username)
	if err != nil {
		user, err = s.userRepo.GetByEmail(req.Username)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		}
	}
	if !user.IsActive {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user is inactive"})
	}
	if err := identityService.ComparePassword(user.PasswordHash, req.Password); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	if s.jwtManager == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "jwt not configured"})
	}

	accessToken, err := s.jwtManager.GenerateToken(user.ID.String(), user.Username, "user", "client", []string{"read:users"})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"access_token": accessToken,
		"refresh_token": "demo-refresh-token",
		"token_type": "Bearer",
		"expires_in": 900,
	})
}


func (s *IdentityService) Refresh(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "refresh"})
}

func (s *IdentityService) Activate(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "activate"})
}

func (s *IdentityService) GetJWKS(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "jwks"})
}

func (s *IdentityService) ListUsers(c echo.Context) error {
	return c.JSON(http.StatusOK, []string{})
}

func (s *IdentityService) CreateUser(c echo.Context) error {
	var req struct {
		Username    string `json:"username"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		SubjectType string `json:"subject_type"`
		ActorType   string `json:"actor_type"`
	}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if len(body) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "empty body"})
	}
	if err := json.Unmarshal(body, &req); err != nil {
		trimmed := string(body)
		trimmed = strings.TrimSpace(trimmed)
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			trimmed = strings.Trim(trimmed, "{}")
			parts := strings.Split(trimmed, ",")
			for _, part := range parts {
				kv := strings.SplitN(part, ":", 2)
				if len(kv) != 2 {
					continue
				}
				key := strings.TrimSpace(strings.Trim(kv[0], `"`))
				value := strings.TrimSpace(strings.Trim(kv[1], `"`))
				switch key {
				case "username":
					req.Username = value
				case "email":
					req.Email = value
				case "password":
					req.Password = value
				case "subject_type":
					req.SubjectType = value
				case "actor_type":
					req.ActorType = value
				}
			}
		} else {
			return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request", "detail": err.Error()})
		}
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "username, email and password are required"})
	}

	hash, err := identityService.HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
	}

	var subject identityModels.SubjectType
	if err := s.db.Where("name = ?", req.SubjectType).First(&subject).Error; err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid subject type"})
	}
	var actor identityModels.ActorType
	if err := s.db.Where("name = ?", req.ActorType).First(&actor).Error; err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid actor type"})
	}

	user := &identityModels.User{
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  hash,
		IsActive:      true,
		SubjectTypeID: subject.ID,
		ActorTypeID:   actor.ID,
	}
	if err := s.userRepo.Create(user); err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "user already exists"})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"id": user.ID,
		"username": user.Username,
		"email": user.Email,
		"is_active": user.IsActive,
	})
}

func (s *IdentityService) GetUser(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "user"})
}

func (s *IdentityService) ActivateUser(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "activated"})
}

func (s *IdentityService) DeactivateUser(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "deactivated"})
}

func (s *IdentityService) DeleteUser(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "deleted"})
}

func (s *IdentityService) AssignRole(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "role assigned"})
}

func (s *IdentityService) RemoveRole(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "role removed"})
}

func (s *IdentityService) ListRoles(c echo.Context) error {
	return c.JSON(http.StatusOK, []string{})
}

func (s *IdentityService) CreateRole(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "role created"})
}

func (s *IdentityService) AssignPermission(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "permission assigned"})
}

func (s *IdentityService) RemovePermission(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "permission removed"})
}

func (s *IdentityService) ListPermissions(c echo.Context) error {
	return c.JSON(http.StatusOK, []string{})
}

func (s *IdentityService) CreatePermission(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "permission created"})
}

func (s *IdentityService) ProvisionUser(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "user provisioned"})
}

func (s *IdentityService) GetUserInternal(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "user"})
}
