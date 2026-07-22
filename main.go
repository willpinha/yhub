package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/spf13/afero"
)

func main() {
	if err := run(); err != nil {
		slog.Error("error while running", "error", err)
		os.Exit(1)
	}
}

func run() error {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
	})))

	return MainCommand(afero.NewOsFs()).Run(context.Background(), os.Args)
}
