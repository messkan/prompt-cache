package logging

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger is the global logger instance
var Logger zerolog.Logger

// Init initializes the global logger
func Init(level string) {
	// Set time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Parse log level
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	// Check if running in development mode
	if os.Getenv("ENV") == "development" || os.Getenv("GIN_MODE") != "release" {
		// Pretty console output for development
		Logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		}).With().Timestamp().Caller().Logger().Level(lvl)
	} else {
		// JSON output for production
		Logger = zerolog.New(os.Stdout).With().Timestamp().Logger().Level(lvl)
	}

	// Set as global logger
	log.Logger = Logger
}

// WithRequestID adds a request ID to the logger context
func WithRequestID(requestID string) zerolog.Logger {
	return Logger.With().Str("request_id", requestID).Logger()
}

// Info logs an info message
func Info() *zerolog.Event {
	return Logger.Info()
}

// Debug logs a debug message
func Debug() *zerolog.Event {
	return Logger.Debug()
}

// Warn logs a warning message
func Warn() *zerolog.Event {
	return Logger.Warn()
}

// Error logs an error message
func Error() *zerolog.Event {
	return Logger.Error()
}

// Fatal logs a fatal message and exits
func Fatal() *zerolog.Event {
	return Logger.Fatal()
}
