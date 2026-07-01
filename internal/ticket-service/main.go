package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"autorent-backend/internal/ticket-service/internal/models"
	"autorent-backend/internal/ticket-service/internal/repository"
	"autorent-backend/internal/ticket-service/internal/service"
	"autorent-backend/pkg/observability"
)

type TicketServiceApp struct {
	db        *gorm.DB
	ticketSvc *service.TicketService
}

func main() {
	_ = godotenv.Load(".env")

	logger, err := observability.InitLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	ctx := context.Background()
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		otelEndpoint = "http://otel-collector:4318"
	}

	tp, err := observability.InitTracer(ctx, "ticket-service", otelEndpoint)
	if err != nil {
		logger.Warn("failed to init tracer", zap.Error(err))
	} else {
		defer tp.Shutdown(ctx)
	}

	db, err := initDatabase()
	if err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}

	if err := db.AutoMigrate(&models.Ticket{}); err != nil {
		logger.Fatal("failed to migrate tickets", zap.Error(err))
	}

	ticketRepo := repository.NewTicketRepository(db)
	ticketSvc := service.NewTicketService(ticketRepo)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	app := &TicketServiceApp{db: db, ticketSvc: ticketSvc}

	e.GET("/healthz", app.Healthz)
	e.GET("/tickets", app.ListTickets)
	e.POST("/tickets", app.CreateTicket)
	e.GET("/tickets/:id", app.GetTicket)
	e.PATCH("/tickets/:id/status", app.UpdateTicketStatus)

	go func() {
		logger.Info("starting ticket-service", zap.String("port", "8080"))
		if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}
}

func initDatabase() (*gorm.DB, error) {
	dsn := os.Getenv("ConnectionStrings__DbConnection")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=ticket_db port=5432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}

func (app *TicketServiceApp) Healthz(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (app *TicketServiceApp) ListTickets(c echo.Context) error {
	tickets, err := app.ticketSvc.ListTickets()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, tickets)
}

func (app *TicketServiceApp) CreateTicket(c echo.Context) error {
	var payload struct {
		Type        models.TicketType `json:"type"`
		Title       string            `json:"title"`
		Description string            `json:"description"`
		RequesterID string            `json:"requester_id"`
		ReferenceID string            `json:"reference_id"`
		Metadata    map[string]any    `json:"metadata"`
	}
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	ticket, err := app.ticketSvc.CreateTicket(struct {
		Type        models.TicketType `json:"type"`
		Title       string            `json:"title"`
		Description string            `json:"description"`
		RequesterID string            `json:"requester_id"`
		ReferenceID string            `json:"reference_id"`
		Metadata    map[string]any    `json:"metadata"`
	}{
		Type:        payload.Type,
		Title:       payload.Title,
		Description: payload.Description,
		RequesterID: payload.RequesterID,
		ReferenceID: payload.ReferenceID,
		Metadata:    payload.Metadata,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, ticket)
}

func (app *TicketServiceApp) GetTicket(c echo.Context) error {
	ticket, err := app.ticketSvc.GetTicket(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "ticket not found"})
	}
	return c.JSON(http.StatusOK, ticket)
}

func (app *TicketServiceApp) UpdateTicketStatus(c echo.Context) error {
	var payload struct {
		Status models.TicketStatus `json:"status"`
	}
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	if payload.Status == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "status is required"})
	}

	ticket, err := app.ticketSvc.UpdateTicketStatus(c.Param("id"), payload.Status)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "ticket not found"})
	}
	return c.JSON(http.StatusOK, ticket)
}
