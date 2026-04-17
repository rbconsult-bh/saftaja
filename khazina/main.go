package main

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/rbconsult-bh/saftaja/khazina/internal/config"
	"github.com/rbconsult-bh/saftaja/khazina/server"
)

func main() {
	ctx := context.Background()

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	deps, err := server.InitDependencies(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize dependencies")
	}
	defer deps.Cleanup()

	r := server.BuildRouter(cfg, deps)

	server.RunServer(ctx, cfg, r)
}
