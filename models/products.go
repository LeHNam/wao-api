package models

import (
	"github.com/LeHNam/wao-api/services/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Product struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string          `gorm:"type:varchar(255);not null" json:"name"`
	Code      string          `gorm:"type:varchar(100);unique;not null" json:"code"`
	Img       string          `gorm:"type:text" json:"img"`
	Options   []ProductOption `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"options"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt *time.Time      `json:"deleted_at,omitempty"`
}

func NewProduct(db *gorm.DB) database.Repository[Product] {
	return database.NewPostgresRepository[Product](db)
}
