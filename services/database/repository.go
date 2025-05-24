package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// contextKey is a custom type for context keys to avoid collisions
type PostgresRepository[T any] struct {
	db *gorm.DB
}

func NewPostgresRepository[T any](db *gorm.DB) Repository[T] {
	return &PostgresRepository[T]{db: db}
}

// Basic CRUD Operations
func (r *PostgresRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *PostgresRepository[T]) CreateMany(ctx context.Context, entities []T) error {
	return r.db.WithContext(ctx).Create(entities).Error
}

func (r *PostgresRepository[T]) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(new(T)).Where("id = ?", id).Updates(updates).Error
}

func (r *PostgresRepository[T]) BatchUpdate(ctx context.Context, updateItems []BatchUpdateItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range updateItems {
			if err := tx.Model(new(T)).
				Where(item.Filter).
				Updates(item.Update).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func (r *PostgresRepository[T]) BatchUpdateLock(ctx context.Context, updateItems []BatchUpdateItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range updateItems {
			if err := tx.Model(new(T)).
				Where(item.Filter).
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Updates(item.Update).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PostgresRepository[T]) Upsert(ctx context.Context, entities []T) error {
	return r.db.WithContext(ctx).Save(entities).Error
}

// Advanced Query Operations
func (r *PostgresRepository[T]) Find(ctx context.Context, conditions map[string]interface{}, selectFields []string, limit, offset int, sort *string) ([]T, error) {
	var entities []T
	query := r.db.WithContext(ctx)

	if len(selectFields) > 0 {
		query = query.Select(selectFields)
	}

	query = applyConditions(query, conditions)

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if sort != nil {
		multiSort := strings.Split(*sort, ",")
		for _, sortField := range multiSort {
			sortField = strings.TrimSpace(sortField)
			isDESCSort := strings.HasPrefix(sortField, "-")
			if isDESCSort {
				field := sortField[1:]
				exists := CheckIfColumnExists(new(T), field)
				if !exists {
					continue
				}
				query = query.Order(field + " DESC")
				continue
			}

			exists := CheckIfColumnExists(new(T), sortField)
			if !exists {
				continue
			}
			query = query.Order(sortField)
		}
	}

	if exists := CheckIfColumnExists(new(T), "deleted_at"); exists {
		query = query.Where("deleted_at IS NULL")
	}

	err := query.Find(&entities).Error
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *PostgresRepository[T]) FindOne(ctx context.Context, conditions map[string]interface{}, selectFields []string) (*T, error) {
	var entity T
	query := r.db.WithContext(ctx)

	if len(selectFields) > 0 {
		query = query.Select(selectFields)
	}

	query = applyConditions(query, conditions)

	if exists := CheckIfColumnExists(new(T), "deleted_at"); exists {
		query = query.Where("deleted_at IS NULL")
	}

	err := query.First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *PostgresRepository[T]) Count(ctx context.Context, conditions map[string]interface{}) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(new(T))

	query = applyConditions(query, conditions)

	err := query.Count(&count).Error
	return count, err
}

// Add these new methods
func (r *PostgresRepository[T]) GetDB() *gorm.DB {
	return r.db
}

func (r *PostgresRepository[T]) WithTx(tx *gorm.DB) Repository[T] {
	return &PostgresRepository[T]{db: tx}
}

func applyConditions(query *gorm.DB, conditions map[string]interface{}) *gorm.DB {
	if len(conditions) > 0 {
		for key, value := range conditions {
			cleanedKey := key
			if cleanedKey == "" || cleanedKey == "OR" {
				continue
			}

			switch {
			case strings.HasSuffix(cleanedKey, CONDITION_IN):
				cleanKey := strings.TrimSuffix(cleanedKey, CONDITION_IN)
				query = query.Where(cleanKey+" IN ?", value)
			case strings.HasSuffix(cleanedKey, CONDITION_NOT_IN):
				cleanKey := strings.TrimSuffix(cleanedKey, CONDITION_NOT_IN)
				query = query.Where(cleanKey+" NOT IN ?", value)
			case strings.HasSuffix(cleanedKey, CONDITION_EXIST_IN_ARRAY_OF_OBJECT):
				cleanKey := strings.TrimSuffix(cleanedKey, CONDITION_EXIST_IN_ARRAY_OF_OBJECT)
				query = query.Where(cleanKey+" @> ?", value)
			case strings.HasSuffix(cleanedKey, CONDITION_BETWEEN_AND):
				if rangeValues, ok := value.([]interface{}); ok && len(rangeValues) == 2 {
					cleanKey := strings.TrimSuffix(cleanedKey, CONDITION_BETWEEN_AND)
					query = query.Where(cleanKey+" BETWEEN ? AND ?", rangeValues[0], rangeValues[1])
				} else {
					log.Printf("Invalid range values for key: %s", cleanedKey)
				}
			case strings.HasSuffix(cleanedKey, CONDITION_LIKE):
				cleanKey := strings.TrimSuffix(cleanedKey, CONDITION_LIKE)
				query = query.Where("LOWER("+cleanKey+") LIKE ?", "%"+strings.ToLower(value.(string))+"%")
			case strings.HasSuffix(cleanedKey, CONDITION_NOT_NULL):
				query = query.Where(cleanedKey)
			case strings.HasSuffix(cleanedKey, CONDITION_NOT_LIKE):
				cleanKey := strings.TrimSuffix(cleanedKey, CONDITION_NOT_LIKE)
				query = query.Where("LOWER("+cleanKey+") NOT LIKE ?", "%"+strings.ToLower(value.(string))+"%")
			case strings.HasSuffix(cleanedKey, CONDITION_EQUAL):
				cleanKey := strings.TrimSuffix(cleanedKey, CONDITION_EQUAL)
				query = query.Where(cleanKey+" = ?", value)
			case strings.HasSuffix(cleanedKey, CONDITION_NOT_EQUAL):
				cleanKey := strings.TrimSuffix(cleanedKey, CONDITION_NOT_EQUAL)
				query = query.Where(cleanKey+" != ?", value)
			case strings.HasSuffix(cleanedKey, CONDITION_GREATER_THAN):
				cleanKey := strings.TrimSuffix(cleanedKey, CONDITION_GREATER_THAN)
				query = query.Where(cleanKey+" > ?", value)
			case strings.HasSuffix(cleanedKey, CONDITION_GREATER_THAN_OR_EQUAL):
				cleanKey := strings.TrimSuffix(cleanedKey, CONDITION_GREATER_THAN_OR_EQUAL)
				query = query.Where(cleanKey+" >= ?", value)
			case strings.HasSuffix(cleanedKey, CONDITION_LESS_THAN):
				cleanKey := strings.TrimSuffix(cleanedKey, CONDITION_LESS_THAN)
				query = query.Where(cleanKey+" < ?", value)
			case strings.HasSuffix(cleanedKey, CONDITION_LESS_THAN_OR_EQUAL):
				cleanKey := strings.TrimSuffix(cleanedKey, CONDITION_LESS_THAN_OR_EQUAL)
				query = query.Where(cleanKey+" <= ?", value)
			default:
				query = query.Where(cleanedKey+" = ?", value)
			}
		}
	}
	if orConditions, ok := conditions["OR"].([]map[string]interface{}); ok && len(orConditions) > 0 {
		orQuery := query.Session(&gorm.Session{NewDB: true})
		for _, cond := range orConditions {
			condition := query.Session(&gorm.Session{NewDB: true})
			condition = applyConditions(condition, cond)
			orQuery = orQuery.Or(condition)
		}

		query = query.Where(orQuery)
	}
	return query
}

func (r *PostgresRepository[T]) First(ctx context.Context, id uuid.UUID) (*T, error) {
	var entity T

	conditions := map[string]interface{}{
		"id": id,
	}

	if exists := CheckIfColumnExists(new(T), "deleted_at"); exists {
		conditions["deleted_at"] = nil
	}

	query := r.db.WithContext(ctx)
	query = applyConditions(query, conditions)

	err := query.First(&entity).Error
	if err != nil {
		return nil, err
	}

	// Convert JSON fields to structs if necessary
	if err := r.convertJSONFields(&entity); err != nil {
		return nil, fmt.Errorf("failed to convert JSON fields: %w", err)
	}

	return &entity, nil
}

func (r *PostgresRepository[T]) FirstWithPreload(ctx context.Context, preloads []PreloadData, id uuid.UUID) (*T, error) {
	var entity T

	conditions := map[string]interface{}{
		"id": id,
	}

	if exists := CheckIfColumnExists(new(T), "deleted_at"); exists {
		conditions["deleted_at"] = nil
	}

	query := r.db.WithContext(ctx)
	for _, preload := range preloads {
		query = query.Preload(preload.Field, preload.Args...)
	}
	query = applyConditions(query, conditions)

	err := query.First(&entity).Error
	if err != nil {
		return nil, err
	}

	// Convert JSON fields to structs if necessary
	if err := r.convertJSONFields(&entity); err != nil {
		return nil, fmt.Errorf("failed to convert JSON fields: %w", err)
	}

	return &entity, nil
}

func (r *PostgresRepository[T]) convertJSONFields(entity *T) error {
	if entity == nil {
		return nil
	}

	// Get the type of the entity
	val := reflect.ValueOf(entity).Elem()
	typ := val.Type()

	// Iterate through the fields of the struct
	for i := 0; i < typ.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Check if the field is of type datatypes.JSON
		if fieldType.Type == reflect.TypeOf(datatypes.JSON{}) {
			// Get the actual JSON data
			jsonData := field.Interface().(datatypes.JSON)
			if len(jsonData) == 0 {
				continue // Skip empty JSON fields
			}

			// Look for a "json" tag to infer the struct field type
			tag := fieldType.Tag.Get("json")
			if tag == "" {
				continue // Skip fields without json tag but don't error
			}

			// Get the field for conversion
			destField := val.FieldByName(fieldType.Name)
			if !destField.IsValid() || !destField.CanSet() {
				continue // Skip invalid fields but don't error
			}

			// Create an instance of the field's type
			destValue := reflect.New(destField.Type()).Interface()

			// Unmarshal JSON into the field
			if err := json.Unmarshal(jsonData, destValue); err != nil {
				return fmt.Errorf("failed to unmarshal JSON for field %s: %w", fieldType.Name, err)
			}

			// Set the unmarshalled value back to the field
			destField.Set(reflect.ValueOf(destValue).Elem())
		}
	}

	return nil
}

func (r *PostgresRepository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	if exists := CheckIfColumnExists(new(T), "deleted_at"); exists {
		updates := map[string]interface{}{
			"deleted_at": time.Now(),
		}
		return r.db.WithContext(ctx).Model(new(T)).Where("id = ?", id).Updates(updates).Error
	}

	return r.DeleteWhere(ctx, map[string]interface{}{"id": id})
}

func (r *PostgresRepository[T]) DeleteWhere(ctx context.Context, conditions map[string]interface{}) error {
	query := r.db.WithContext(ctx)

	query = applyConditions(query, conditions)

	return query.Delete(new(T)).Error
}

func (r *PostgresRepository[T]) Paginate(ctx context.Context, conditions map[string]interface{}, selectFields []string, limit, offset int, sort string) Pagination[T] {
	var entities []T
	var totalCount int64

	// Base query
	query := r.db.WithContext(ctx)

	// Apply field selection
	if len(selectFields) > 0 {
		query = query.Select(selectFields)
	}

	// Apply conditions
	if len(conditions) > 0 {
		query = applyConditions(query, conditions)
	}

	if exists := CheckIfColumnExists(new(T), "deleted_at"); exists {
		query = query.Where("deleted_at IS NULL")
	}

	// Get total count before applying pagination
	countQuery := query.Session(&gorm.Session{})
	countQuery.Model(new(T)).Count(&totalCount)

	// Apply pagination (limit and offset)
	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	// Apply sorting
	if sort != "" {
		if strings.HasPrefix(sort, "-") {
			column := strings.TrimPrefix(sort, "-")
			query = query.Order(fmt.Sprintf(`"%s" DESC`, column))
		} else {
			query = query.Order(fmt.Sprintf(`"%s" ASC`, sort))
		}
	} else {
		if exists := CheckIfColumnExists(new(T), "created_at"); exists {
			query = query.Order("created_at desc")
		} else if exists := CheckIfColumnExists(new(T), "created_date"); exists {
			query = query.Order("created_date desc")
		}
	}

	// Fetch paginated results
	query.Find(&entities)

	// Calculate page and total pages, handling division by zero
	var page, totalPages int
	if limit > 0 {
		page = offset/limit + 1
		totalPages = int(math.Ceil(float64(totalCount) / float64(limit)))
	} else {
		page = 1
		totalPages = 1
	}

	return Pagination[T]{
		Total:     totalCount,
		Items:     entities,
		Limit:     limit,
		Page:      page,
		TotalPage: totalPages,
	}
}

// BulkDelete soft-deletes multiple entities by ID
func (r *PostgresRepository[T]) BulkDelete(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	if exists := CheckIfColumnExists(new(T), "deleted_at"); exists {
		updates := map[string]interface{}{
			"deleted_at": time.Now(),
		}
		return r.db.WithContext(ctx).Model(new(T)).Where("id IN ?", ids).Updates(updates).Error
	}

	return r.db.WithContext(ctx).Delete(new(T), "id IN ?", ids).Error
}

// checkColumnExistsCache is a cache for column existence checks
var (
	checkColumnExistsCache = make(map[string]bool)
	columnCacheMutex       = &sync.RWMutex{}
)

// CheckIfColumnExists checks if a column exists in the entity
func CheckIfColumnExists(entity interface{}, columnName string) bool {
	// Get the type of the entity
	entityType := reflect.TypeOf(entity).Elem()
	cacheKey := entityType.String() + ":" + columnName

	// Check cache first with read lock
	columnCacheMutex.RLock()
	exists, found := checkColumnExistsCache[cacheKey]
	columnCacheMutex.RUnlock()

	if found {
		return exists
	}

	// Not in cache, check the entity
	val := reflect.ValueOf(entity).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Tag.Get("json") == columnName {
			// Update cache with write lock
			columnCacheMutex.Lock()
			checkColumnExistsCache[cacheKey] = true
			columnCacheMutex.Unlock()
			return true
		}
	}

	// Update cache with write lock
	columnCacheMutex.Lock()
	checkColumnExistsCache[cacheKey] = false
	columnCacheMutex.Unlock()
	return false
}

// BatchCreate is a more efficient way to insert multiple records
func (r *PostgresRepository[T]) BatchCreate(ctx context.Context, entities []T, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	return r.db.WithContext(ctx).CreateInBatches(entities, batchSize).Error
}

// ExecuteBulkOps executes any bulk operations with a callback
func (r *PostgresRepository[T]) ExecuteBulkOps(ctx context.Context, callback func(*gorm.DB) error) error {
	return callback(r.db.WithContext(ctx))
}

// WithContextTimeout creates a new context with timeout and returns a repository with it
func (r *PostgresRepository[T]) WithContextTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc, Repository[T]) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	return ctxWithTimeout, cancel, &PostgresRepository[T]{db: r.db.WithContext(ctxWithTimeout)}
}

// FindOneWithTimeout finds a single entity with context timeout
func (r *PostgresRepository[T]) FindOneWithTimeout(ctx context.Context, conditions map[string]interface{}, selectFields []string, timeout time.Duration) (*T, error) {
	ctxWithTimeout, cancel, repo := r.WithContextTimeout(ctx, timeout)
	defer cancel()
	return repo.(*PostgresRepository[T]).FindOne(ctxWithTimeout, conditions, selectFields)
}

// IsRecordNotFoundError checks if the error is a record not found error
func IsRecordNotFoundError(err error) bool {
	return err != nil && errors.Is(err, gorm.ErrRecordNotFound)
}

// FindOneOrNil returns entity or nil if not found without error
func (r *PostgresRepository[T]) FindOneOrNil(ctx context.Context, conditions map[string]interface{}, selectFields []string) (*T, error) {
	result, err := r.FindOne(ctx, conditions, selectFields)
	if IsRecordNotFoundError(err) {
		return nil, nil
	}
	return result, err
}

// Transaction executes operations in a database transaction
func (r *PostgresRepository[T]) Transaction(ctx context.Context, fn func(Repository[T]) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := r.WithTx(tx)
		return fn(txRepo)
	})
}

// FindByIDs retrieves entities matching the provided slice of IDs
func (r *PostgresRepository[T]) FindByIDs(ctx context.Context, ids []uuid.UUID, selectFields []string) ([]T, error) {
	if len(ids) == 0 {
		return []T{}, nil
	}

	conditions := map[string]interface{}{
		"id IN": ids,
	}

	return r.Find(ctx, conditions, selectFields, 0, 0, nil)
}

// PaginateWithPreload retrieves paginated results with preloaded relations
func (r *PostgresRepository[T]) PaginateWithPreload(
	ctx context.Context,
	preloads []PreloadData,
	conditions map[string]interface{},
	selectFields []string,
	limit, offset int,
	sort string,
) Pagination[T] {
	var entities []T
	var totalCount int64

	// Base query
	query := r.db.WithContext(ctx)

	// Apply field selection
	if len(selectFields) > 0 {
		query = query.Select(selectFields)
	}

	// Apply preloading
	for _, preload := range preloads {
		query = query.Preload(preload.Field, preload.Args...)
	}

	// Apply conditions
	if len(conditions) > 0 {
		query = applyConditions(query, conditions)
	}

	if exists := CheckIfColumnExists(new(T), "deleted_at"); exists {
		query = query.Where("deleted_at IS NULL")
	}

	// Get total count before applying pagination
	countQuery := query.Session(&gorm.Session{})
	countQuery.Model(new(T)).Count(&totalCount)

	// Apply pagination (limit and offset)
	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	// Apply sorting
	if sort != "" {
		multiSort := strings.Split(sort, ",")
		for _, sortField := range multiSort {
			sortField = strings.TrimSpace(sortField)
			isDESCSort := strings.HasPrefix(sortField, "-")
			if isDESCSort {
				field := sortField[1:]
				exists := CheckIfColumnExists(new(T), field)
				if !exists {
					continue
				}
				query = query.Order(field + " DESC")
			} else {
				exists := CheckIfColumnExists(new(T), sortField)
				if !exists {
					continue
				}
				query = query.Order(sortField)
			}
		}
	} else {
		if exists := CheckIfColumnExists(new(T), "created_at"); exists {
			query = query.Order("created_at desc")
		} else if exists := CheckIfColumnExists(new(T), "created_date"); exists {
			query = query.Order("created_date desc")
		}
	}

	// Fetch paginated results
	query.Find(&entities)

	// Calculate page and total pages, handling division by zero
	var page, totalPages int
	if limit > 0 {
		page = offset/limit + 1
		totalPages = int(math.Ceil(float64(totalCount) / float64(limit)))
	} else {
		page = 1
		totalPages = 1
	}

	return Pagination[T]{
		Total:     totalCount,
		Items:     entities,
		Limit:     limit,
		Page:      page,
		TotalPage: totalPages,
	}
}

// Basic CRUD Operations
func (r *PostgresRepository[T]) CreateAndReturn(ctx context.Context, entity *T) (T, error) {
	result := r.db.WithContext(ctx).Create(entity)
	return *entity, result.Error
}

// FindByID retrieves entity matching the provided of ID
func (r *PostgresRepository[T]) FindByID(ctx context.Context, id uuid.UUID, selectFields []string) (*T, error) {
	if id == uuid.Nil {
		return nil, nil
	}

	conditions := map[string]interface{}{
		"id": id,
	}

	return r.FindOne(ctx, conditions, selectFields)
}

func (r *PostgresRepository[T]) UpdateFields(ctx context.Context, conditions map[string]interface{}, updates map[string]interface{}) error {
	query := r.db.WithContext(ctx).Model(new(T))

	query = applyConditions(query, conditions)

	return query.Updates(updates).Error
}

func (r *PostgresRepository[T]) FindWithJoinAndPreload(ctx context.Context, conditions map[string]interface{}, selectFields []string, limit, offset int, sort *string, joins []string, preloads []PreloadData) ([]T, error) {
	var entities []T
	query := r.db.WithContext(ctx)

	if len(selectFields) > 0 {
		query = query.Select(selectFields)
	}

	// Apply joins
	for _, join := range joins {
		query = query.Joins(join)
	}

	// Apply preloads
	for _, preload := range preloads {
		query = query.Preload(preload.Field, preload.Args...)
	}

	query = applyConditions(query, conditions)

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if sort != nil {
		multiSort := strings.Split(*sort, ",")
		for _, sortField := range multiSort {
			sortField = strings.TrimSpace(sortField)
			isDESCSort := strings.HasPrefix(sortField, "-")
			if isDESCSort {
				field := sortField[1:]
				exists := CheckIfColumnExists(new(T), field)
				if !exists {
					continue
				}
				query = query.Order(field + " DESC")
				continue
			}

			exists := CheckIfColumnExists(new(T), sortField)
			if !exists {
				continue
			}
			query = query.Order(sortField)
		}
	}

	tableName := getTableNameByStruct[T]()
	if exists := CheckIfColumnExists(new(T), "deleted_at"); exists {
		query = query.Where(tableName + ".deleted_at IS NULL")
	}

	err := query.Find(&entities).Error
	if err != nil {
		return nil, err
	}

	return entities, nil
}
func getTableNameByStruct[T any]() string {
	// Use reflection to get the name of the type
	return reflect.TypeOf((*T)(nil)).Elem().Name()
}

func (r *PostgresRepository[T]) CountWithJoin(ctx context.Context, conditions map[string]interface{}, joins []string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(new(T))
	for _, join := range joins {
		query = query.Joins(join)
	}

	query = applyConditions(query, conditions)

	err := query.Count(&count).Error
	return count, err
}
