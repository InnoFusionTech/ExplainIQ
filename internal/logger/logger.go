package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Config holds logger configuration
type Config struct {
	Level  string
	Format string // "json" or "text"
}

// New creates a new logger instance with consistent configuration
func New(config ...Config) *logrus.Logger {
	logger := logrus.New()

	// Apply default config
	cfg := Config{
		Level:  "info",
		Format: "text",
	}
	if len(config) > 0 {
		cfg = config[0]
	}

	// Set log level from environment or config
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = cfg.Level
	}

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Set formatter
	if cfg.Format == "json" || os.Getenv("LOG_FORMAT") == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	return logger
}











