package models

import (
	"github.com/LeHNam/wao-api/services/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

// User model
type User struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	Name      string     `json:"name" `
	Email     string     `json:"email"`
	Username  string     `json:"username" `
	Password  string     `json:"password"`
	Role      string     `json:"role"`
	Token     string     `json:"token" bson:"token"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedBy uuid.UUID  `json:"created_by" gorm:"type:uuid;"`
	UpdatedBy uuid.UUID  `json:"updated_by" gorm:"type:uuid;"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

func NewUser(db *gorm.DB) database.Repository[User] {
	return database.NewPostgresRepository[User](db)
}
