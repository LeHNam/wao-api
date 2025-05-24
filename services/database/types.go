package database

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// QueryOption represents a function that modifies a GORM query
type QueryOption func(*gorm.DB) *gorm.DB

type PreloadData struct {
	Field string
	Args  []interface{}
}
type Pagination[T any] struct {
	Total     int64 `json:"total"`
	Items     []T   `json:"items"`
	Limit     int   `json:"limit"`
	Page      int   `json:"page"`
	TotalPage int   `json:"total_page"`
}

type BatchUpdateItem struct {
	Filter map[string]interface{}
	Update map[string]interface{}
}

type Repository[T any] interface {
	GetDB() *gorm.DB
	WithTx(tx *gorm.DB) Repository[T]
	Transaction(ctx context.Context, fn func(Repository[T]) error) error
	Find(ctx context.Context, conditions map[string]interface{}, selectFields []string, limit, offset int, sort *string) ([]T, error)
	Count(ctx context.Context, conditions map[string]interface{}) (int64, error)
	Create(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id uuid.UUID) error
	First(ctx context.Context, id uuid.UUID) (*T, error)
	Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	DeleteWhere(ctx context.Context, conditions map[string]interface{}) error
	CreateMany(ctx context.Context, entities []T) error
	UpdateFields(ctx context.Context, conditions map[string]interface{}, updates map[string]interface{}) error
	FindOne(ctx context.Context, conditions map[string]interface{}, selectFields []string) (*T, error)
	FindWithJoinAndPreload(ctx context.Context, conditions map[string]interface{}, selectFields []string, limit, offset int, sort *string, joins []string, preloads []PreloadData) ([]T, error)
	CountWithJoin(ctx context.Context, conditions map[string]interface{}, joins []string) (int64, error)
}
