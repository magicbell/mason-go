// Package logger provides a convenience function to constructing a logger
// for use. This is required not just for applications but for testing.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New constructs a Sugared Logger that writes to stdout and
// provides human-readable timestamps.
func New(service string, env string, outputPaths ...string) (*zap.SugaredLogger, error) {
	switch env {
	case "production":
		return newProduction(service, "stdout")
	default:
		return newDevelopment(service, outputPaths...)
	}
}

func newProduction(service string, outputPaths ...string) (*zap.SugaredLogger, error) {
	config := zap.NewProductionConfig()

	if outputPaths != nil {
		config.OutputPaths = outputPaths
	}

	log, err := config.Build(zap.WithCaller(true))
	if err != nil {
		return nil, err
	}

	return log.Sugar(), nil
}

func newDevelopment(service string, outputPaths ...string) (*zap.SugaredLogger, error) {
	config := zap.NewDevelopmentConfig()

	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.InitialFields = map[string]any{
		"service": service,
	}

	config.OutputPaths = []string{"stdout"}
	if outputPaths != nil {
		config.OutputPaths = outputPaths
	}

	log, err := config.Build(zap.WithCaller(true))
	if err != nil {
		return nil, err
	}

	return log.Sugar(), nil
}
