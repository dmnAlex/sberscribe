package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

func Init(level string) {
	opts := &slog.HandlerOptions{
		Level: logLevelFromString(level),
	}

	Log = slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func logLevelFromString(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
