package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/LeHNam/wao-api/config"
)

func NewZapLogger(cfg *config.Config) *zap.Logger {
	// Determine environment from environment variable
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	// Initialize Zap configuration based on environment
	var zapConfig zap.Config

	// Set default level to InfoLevel
	level := zap.InfoLevel

	// Configure based on environment
	if env == "production" || env == "prod" {
		// Configuration for production environment
		zapConfig = zap.NewProductionConfig()
	} else {
		// Configuration for development environment
		zapConfig = zap.NewDevelopmentConfig()
		// Display colorized level in development environment
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		// Use debug level in development
		level = zap.DebugLevel
	}

	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Initialize logger
	logger, err := zapConfig.Build()
	if err != nil {
		// If unable to initialize logger, use default logger and log the error
		defaultLogger, _ := zap.NewProduction()
		defaultLogger.Error("Failed to initialize custom logger", zap.Error(err))
		return defaultLogger
	}

	return logger
}

// SugaredLogger returns a SugaredLogger from zap.Logger
func SugaredLogger(logger *zap.Logger) *zap.SugaredLogger {
	return logger.Sugar()
}
