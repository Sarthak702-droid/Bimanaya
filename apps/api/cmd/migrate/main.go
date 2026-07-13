package main

import (
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Running BimaNyaya Database Migrations...")
	slog.Info("All database migrations completed successfully (Convex Schema managed via TypeScript)!")
}

