package limiter

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"time"
)

var RedisClient *redis.Client

func InitRedisClient(addr, password string, db int) {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := RedisClient.Ping(ctx).Err()
	if err != nil {
		log.Fatal().Msgf("failed to connect to Redis: %s", err.Error())
	}

	log.Info().Msg("Connected to Redis")
}
