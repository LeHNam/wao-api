package models

import (
	"github.com/LeHNam/wao-api/services/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type ProductOption struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ProductID uuid.UUID `gorm:"type:uuid;index;not null" json:"product_id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Code      string    `gorm:"type:varchar(100);not null" json:"code"`
	Quantity  int       `gorm:"default:0" json:"quantity"`
	Price     float64   `gorm:"type:decimal(10,2);default:0" json:"price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewProductOption(db *gorm.DB) database.Repository[ProductOption] {
	return database.NewPostgresRepository[ProductOption](db)
}
