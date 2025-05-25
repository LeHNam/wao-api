package models

import (
	"time"

	"github.com/LeHNam/wao-api/services/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PurchaseOrderItem struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	PurchaseOrderID   uuid.UUID      `json:"purchase_order_id" gorm:"not null;index"`
	ProductID         uuid.UUID      `json:"product_id" gorm:"not null;index"`
	ProductOptionID   uuid.UUID      `json:"product_option_id" gorm:"not null;index"`
	ProductName       string         `json:"product_name" gorm:"not null"`
	ProductOptionName string         `json:"product_option_name" gorm:"not null"`
	UnitPrice         float64        `json:"unit_price" gorm:"not null"`
	TotalPrice        float64        `json:"total_price" gorm:"not null"`
	Currency          string         `json:"currency" gorm:"not null"`
	Quantity          int            `json:"quantity" gorm:"not null"`
	CreatedAt         time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedBy         uuid.UUID      `json:"created_by" gorm:"type:uuid;"`
	UpdatedBy         uuid.UUID      `json:"updated_by" gorm:"type:uuid;"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

func NewPurchaseOrderItem(db *gorm.DB) database.Repository[PurchaseOrderItem] {
	return database.NewPostgresRepository[PurchaseOrderItem](db)
}
