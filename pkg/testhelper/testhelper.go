package testhelper

import (
	"log/slog"
	"os"
)

func SetupTestLogger(verbose bool) {
	level := slog.LevelWarn
	if verbose {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))
}
