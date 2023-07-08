package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"rate-limiter/limiter"
	"time"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	cfg := defaultConfig()
	log.Debug().Any("config", cfg).Msg("config loaded")

	limiter.InitRedisClient(cfg.RedisConfig.Addr(), cfg.RedisConfig.Password, cfg.RedisConfig.DB)

	err := limiter.LoadRules("limiter_rules.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load rules")
	}

	limiter.ReloadRulesPeriodically("limiter_rules.yaml", 10*time.Second)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(limiter.RateLimiterMiddleware)

	r.Post("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, World!")
	})

	log.Info().Msgf("Starting server at %s", cfg.Server.Addr())
	err = http.ListenAndServe(cfg.Server.Addr(), r)
}
