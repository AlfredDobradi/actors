package logging

import (
	"log/slog"
	"os"
)

var (
	LogLevel  = slog.LevelDebug
	LogFormat = "json"
)

func Init() {
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		switch level {
		case "debug":
			LogLevel = slog.LevelDebug
		case "info":
			LogLevel = slog.LevelInfo
		case "error":
			LogLevel = slog.LevelError
		default:
			slog.Warn("Invalid LOG_LEVEL, defaulting to debug", "providedLevel", level)
			LogLevel = slog.LevelDebug
		}
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		LogFormat = format
	}
	var handler slog.Handler
	switch LogFormat {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: LogLevel,
		})
	case "text":
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: LogLevel,
		})
	default:
		slog.Warn("Invalid LOG_FORMAT, defaulting to json", "providedFormat", LogFormat)
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: LogLevel,
		})
	}

	slog.SetDefault(
		slog.New(handler),
	)
}
