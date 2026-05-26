package logging

import (
	"log/slog"
	"os"

	"github.com/alfreddobradi/actors/pkg/config"
)

func Init() {
	loggingConfig := config.GetConfig().Logging
	level := slog.LevelInfo

	switch loggingConfig.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "error":
		level = slog.LevelError
	default:
		slog.Warn("Invalid log level in config", "provided", loggingConfig.Level)
		level = slog.LevelDebug
	}

	var handler slog.Handler
	switch loggingConfig.Format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level,
			AddSource: loggingConfig.AddSource,
		})
	case "text":
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level,
			AddSource: loggingConfig.AddSource,
		})
	default:
		slog.Warn("Invalid log format in config, defaulting to json", "provided", loggingConfig.Format)
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level,
			AddSource: loggingConfig.AddSource,
		})
	}

	slog.SetDefault(
		slog.New(handler),
	)
}
