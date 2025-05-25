package errors

import (
	"fmt"
	"log"
)

// ErrorLevel represents the severity of an error
type ErrorLevel int

const (
	// Info is for informational messages that don't indicate an error
	Info ErrorLevel = iota
	// Warning is for non-critical errors that don't prevent the application from functioning
	Warning
	// Error is for errors that affect functionality but don't require termination
	Error
	// Fatal is for critical errors that require application termination
	Fatal
)

// AppError represents an application error with context
type AppError struct {
	Message string
	Level   ErrorLevel
	Err     error
}

// String returns a string representation of the error
func (e *AppError) String() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Error implements the error interface
func (e *AppError) Error() string {
	return e.String()
}

// New creates a new AppError
func New(message string, level ErrorLevel, err error) *AppError {
	// Log the error
	logError(message, level, err)

	return &AppError{
		Message: message,
		Level:   level,
		Err:     err,
	}
}

// logError logs the error with appropriate level
func logError(message string, level ErrorLevel, err error) {
	var levelStr string

	switch level {
	case Info:
		levelStr = "INFO"
	case Warning:
		levelStr = "WARNING"
	case Error:
		levelStr = "ERROR"
	case Fatal:
		levelStr = "FATAL"
	}

	if err != nil {
		log.Printf("[%s] %s: %v", levelStr, message, err)
	} else {
		log.Printf("[%s] %s", levelStr, message)
	}
}

// HandleError handles an error based on its level
func HandleError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	// Default to Error level for unknown errors
	return New("An unexpected error occurred", Error, err)
}

// IsFatal checks if an error is fatal
func IsFatal(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Level == Fatal
	}
	return false
}
