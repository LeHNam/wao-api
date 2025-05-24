package database

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func IsNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate key")
}

func IsForeignKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "foreign key constraint")
}

func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "connection refused") ||
		strings.Contains(strings.ToLower(err.Error()), "connection timeout")
}

func IsDeadlockError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "deadlock")
}

func IsConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "constraint violation") ||
		strings.Contains(strings.ToLower(err.Error()), "check constraint")
}

func HandleDBError(err error) error {
	fmt.Print(err)
	if err == nil {
		return errors.New("unknown error")
	}

	switch {
	case IsNotFoundError(err):
		return errors.New("record not found")
	case IsDuplicateKeyError(err):
		return errors.New("record already exists")
	case IsForeignKeyError(err):
		return errors.New("invalid foreign key reference")
	case IsConnectionError(err):
		return errors.New("database connection error")
	case IsDeadlockError(err):
		return errors.New("database deadlock detected")
	case IsConstraintError(err):
		return errors.New("constraint violation")
	default:
		return errors.New("database error")
	}
}
