package models

import (
	"time"

	"github.com/LeHNam/wao-api/services/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PurchaseOrder struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	OrderNumber string         `json:"order_number" gorm:"not null;uniqueIndex"`
	Status      string         `json:"status" gorm:"not null"`
	OrderDate   time.Time      `json:"order_date" gorm:"not null"`
	TotalAmount float64        `json:"total_amount" gorm:"not null"`
	Currency    string         `json:"currency" gorm:"not null"`
	Timezone    string         `json:"timezone" gorm:"not null"`
	Notes       *string        `json:"notes,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedBy   uuid.UUID      `json:"created_by" gorm:"type:uuid;"`
	UpdatedBy   uuid.UUID      `json:"updated_by" gorm:"type:uuid;"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

func NewPurchaseOrder(db *gorm.DB) database.Repository[PurchaseOrder] {
	return database.NewPostgresRepository[PurchaseOrder](db)
}
