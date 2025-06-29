package redis

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

var (
	Client *redis.Client
	Ctx    = context.Background()
)

func InitRedis() {
	Client = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOSTPORT"), // host:port only
		Username: os.Getenv("REDIS_USERNAME"), // "default"
		Password: os.Getenv("REDIS_PASSWORD"), // your password
		DB:       0,
	})

	// Test connection
	_, err := Client.Ping(Ctx).Result()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}
	fmt.Println("✅ Connected to Redis")
}
