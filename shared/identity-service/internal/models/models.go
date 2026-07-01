package models

import (
	"time"

	"github.com/google/uuid"
)

// SubjectType defines the type of subject (user, service, api_key, system)
type SubjectType struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ActorType defines the role of actor (client, partner, admin, internal)
type ActorType struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// User represents a user in the system
type User struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	Username     string          `gorm:"uniqueIndex;not null" json:"username"`
	Email        string          `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string          `gorm:"not null" json:"-"`
	IsActive     bool            `gorm:"default:false" json:"is_active"`
	SubjectTypeID uuid.UUID      `gorm:"type:uuid;not null" json:"subject_type_id"`
	SubjectType  *SubjectType    `gorm:"foreignKey:SubjectTypeID" json:"subject_type,omitempty"`
	ActorTypeID  uuid.UUID       `gorm:"type:uuid;not null" json:"actor_type_id"`
	ActorType    *ActorType      `gorm:"foreignKey:ActorTypeID" json:"actor_type,omitempty"`
	Roles        []Role          `gorm:"many2many:user_roles" json:"roles,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// Role represents a role in the system
type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Users       []User    `gorm:"many2many:user_roles" json:"users,omitempty"`
	Permissions []Permission `gorm:"many2many:role_permissions" json:"permissions,omitempty"`
	ParentRoles []Role    `gorm:"many2many:role_inheritance;foreignKey:ID;joinForeignKey:ChildRoleID;References:ID;joinReferences:ParentRoleID" json:"parent_roles,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Permission represents a permission in the system
type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Description string    `json:"description"`
	Roles       []Role    `gorm:"many2many:role_permissions" json:"roles,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RefreshToken represents a refresh token
type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash string    `gorm:"uniqueIndex;not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// ActivationToken represents an activation token
type ActivationToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash string    `gorm:"uniqueIndex;not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

// UserProvisionRequest represents a user provision request
type UserProvisionRequest struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	UserID       uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
	RequestKey   string          `gorm:"uniqueIndex;not null" json:"request_key"`
	Email        string          `json:"email"`
	SubjectType  string          `json:"subject_type"`
	ActorType    string          `json:"actor_type"`
	CreatedAt    time.Time       `json:"created_at"`
}

// UserRoles join table
type UserRole struct {
	UserID uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	RoleID uuid.UUID `gorm:"type:uuid;primaryKey" json:"role_id"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

// RolePermissions join table
type RolePermission struct {
	RoleID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"role_id"`
	PermissionID uuid.UUID `gorm:"type:uuid;primaryKey" json:"permission_id"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

// RoleInheritance join table for role hierarchy
type RoleInheritance struct {
	ChildRoleID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"child_role_id"`
	ParentRoleID uuid.UUID `gorm:"type:uuid;primaryKey" json:"parent_role_id"`
}

func (RoleInheritance) TableName() string {
	return "role_inheritance"
}
