package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config конфигурация приложения
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Port            string        `mapstructure:"port"`
	Host            string        `mapstructure:"host"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig конфигурация базы данных (postgresql)
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// LogConfig конфигурация логгера
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Configuration priority (highest to lowest):
// 1. Environment variables with APP_ prefix (APP_DATABASE_HOST, APP_SERVER_PORT, etc.)
// 2. .env file in root directory (POSTGRES_HOST=postgres, SERVER_PORT=8080, etc.)
// 3. Default values (hardcoded in setDefaults)

func Load() (*Config, error) {
	v := viper.New()

	setDefaults(v)

	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading .env file: %w", err)
		}
	}

	// Map environment variables to config keys
	// Database
	v.BindEnv("database.host", "POSTGRES_HOST")
	v.BindEnv("database.port", "POSTGRES_PORT")
	v.BindEnv("database.user", "POSTGRES_USER")
	v.BindEnv("database.password", "POSTGRES_PASSWORD")
	v.BindEnv("database.dbname", "POSTGRES_DB")
	v.BindEnv("database.sslmode", "POSTGRES_SSLMODE")

	// Server
	v.BindEnv("server.port", "SERVER_PORT")
	v.BindEnv("server.host", "SERVER_HOST")
	v.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	v.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
	v.BindEnv("server.shutdown_timeout", "SERVER_SHUTDOWN_TIMEOUT")

	// Log
	v.BindEnv("log.level", "LOG_LEVEL")
	v.BindEnv("log.format", "LOG_FORMAT")

	v.AutomaticEnv()
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.read_timeout", 10*time.Second)
	v.SetDefault("server.write_timeout", 10*time.Second)
	v.SetDefault("server.shutdown_timeout", 5*time.Second)

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.dbname", "pr_reviewer")
	v.SetDefault("database.sslmode", "disable")

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
}

func validate(cfg *Config) error {
	if cfg.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	if cfg.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if cfg.Database.Port <= 0 || cfg.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", cfg.Database.Port)
	}

	if cfg.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[strings.ToLower(cfg.Log.Level)] {
		return fmt.Errorf("invalid log level: %s", cfg.Log.Level)
	}

	validLogFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validLogFormats[strings.ToLower(cfg.Log.Format)] {
		return fmt.Errorf("invalid log format: %s", cfg.Log.Format)
	}

	return nil
}

// GetDSN строка подключения к PostgreSQL
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}
