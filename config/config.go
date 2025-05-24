package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration
type Config struct {
	Database struct {
		Host     string `mapstructure:"host" yaml:"host"`
		Port     string `mapstructure:"port" yaml:"port"`
		Username string `mapstructure:"username" yaml:"username"`
		Password string `mapstructure:"password" yaml:"password"`
		DbName   string `mapstructure:"db_name" yaml:"db_name"`
		Ssl      bool   `mapstructure:"ssl" yaml:"ssl"`
	} `mapstructure:"database" yaml:"database"`
	Server struct {
		Port string `mapstructure:"port" yaml:"port"`
	} `mapstructure:"server" yaml:"server"`
	JWT struct {
		Secret    string        `mapstructure:"secret" yaml:"secret"`
		ExpiredAt time.Duration `mapstructure:"expires_at" yaml:"expires_at"`
	} `mapstructure:"jwt" yaml:"jwt"`
}

var config Config

// LoadConfig reads configuration from file or environment variables
func LoadConfig() (*Config, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	// Get the absolute path to the config directory
	configPath, err := filepath.Abs("./config")
	if err != nil {
		return nil, fmt.Errorf("error getting config path: %v", err)
	}

	// Set defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "5432")
	viper.SetDefault("database.username", "admin")
	viper.SetDefault("database.password", "admin")
	viper.SetDefault("database.db_name", "wao")
	viper.SetDefault("server.port", "8080")

	viper.SetConfigType("yaml")
	viper.SetConfigName(env)
	viper.AddConfigPath(configPath)
	viper.AddConfigPath(".")

	// Enable environment variables
	viper.AutomaticEnv()

	// Replace ENV
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Try to read the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file: %v", err)
		return nil, err
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %v", err)
	}

	fmt.Println("The app is running on " + env)
	return &config, nil
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	return &config
}
