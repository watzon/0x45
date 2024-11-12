package ratelimit

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimiter manages both global and per-IP rate limiting
type RateLimiter struct {
	// Redis-based limiter (for prefork mode)
	redis *redis.Client

	// In-memory limiters (for single process mode)
	globalLimiter *rate.Limiter
	ipLimiters    sync.Map

	config   Config
	useRedis bool
	logger   *zap.Logger
}

// Config holds configuration for rate limiting
type Config struct {
	Global struct {
		Enabled bool
		Rate    float64
		Burst   int
	}
	PerIP struct {
		Enabled bool
		Rate    float64
		Burst   int
	}
	Redis    *redis.Client // Optional: only required for prefork mode
	UseRedis bool          // Whether to use Redis (true if prefork is enabled)
}

// New creates a new RateLimiter instance
func New(config Config) *RateLimiter {
	logger, _ := zap.NewProduction()
	if config.UseRedis && config.Redis == nil {
		logger.Panic("Redis client is required when UseRedis is true")
	}

	r := &RateLimiter{
		redis:    config.Redis,
		useRedis: config.UseRedis,
		config:   config,
		logger:   logger,
	}

	// Initialize in-memory limiters if not using Redis
	if !config.UseRedis {
		r.globalLimiter = rate.NewLimiter(rate.Limit(config.Global.Rate), config.Global.Burst)
	}

	return r
}

// Check checks both global and IP-based rate limits
func (r *RateLimiter) Check(ip string) error {
	if r.useRedis {
		return r.checkRedis(ip)
	}
	return r.checkMemory(ip)
}

// checkMemory implements in-memory rate limiting using golang.org/x/time/rate
func (r *RateLimiter) checkMemory(ip string) error {
	// Check global rate limit if enabled
	if r.config.Global.Enabled {
		if !r.globalLimiter.Allow() {
			return fiber.NewError(
				fiber.StatusTooManyRequests,
				"Server is experiencing high load, please try again later",
			)
		}
	}

	// Check IP-specific rate limit if enabled
	if r.config.PerIP.Enabled {
		ipLimiter := r.getIPLimiter(ip)
		if !ipLimiter.Allow() {
			return fiber.NewError(
				fiber.StatusTooManyRequests,
				"Rate limit exceeded, please try again later",
			)
		}
	}

	return nil
}

// getIPLimiter returns a rate limiter for the specified IP address
func (r *RateLimiter) getIPLimiter(ip string) *rate.Limiter {
	limiter, exists := r.ipLimiters.Load(ip)
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(r.config.PerIP.Rate), r.config.PerIP.Burst)
		r.ipLimiters.Store(ip, limiter)
	}
	return limiter.(*rate.Limiter)
}

// checkRedis implements Redis-based rate limiting for prefork mode
func (r *RateLimiter) checkRedis(ip string) error {
	if r.redis == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Redis required for rate limiting in prefork mode")
	}

	ctx := context.Background()

	// Check global rate limit if enabled
	if r.config.Global.Enabled {
		allowed, err := r.checkRedisLimit(ctx, "global", r.config.Global.Rate, r.config.Global.Burst)
		if err != nil {
			r.logger.Error("global rate limit check failed",
				zap.Error(err),
				zap.Float64("rate", r.config.Global.Rate),
				zap.Int("burst", r.config.Global.Burst),
			)
			return fiber.NewError(fiber.StatusInternalServerError, "Rate limit check failed")
		}
		if !allowed {
			return fiber.NewError(
				fiber.StatusTooManyRequests,
				"Server is experiencing high load, please try again later",
			)
		}
	}

	// Check IP-specific rate limit if enabled
	if r.config.PerIP.Enabled {
		allowed, err := r.checkRedisLimit(ctx, fmt.Sprintf("ip:%s", ip), r.config.PerIP.Rate, r.config.PerIP.Burst)
		if err != nil {
			r.logger.Error("IP rate limit check failed",
				zap.Error(err),
				zap.String("ip", ip),
				zap.Float64("rate", r.config.PerIP.Rate),
				zap.Int("burst", r.config.PerIP.Burst),
			)
			return fiber.NewError(fiber.StatusInternalServerError, "Rate limit check failed")
		}
		if !allowed {
			return fiber.NewError(
				fiber.StatusTooManyRequests,
				"Rate limit exceeded, please try again later",
			)
		}
	}

	return nil
}

// checkRedisLimit implements a Redis-based token bucket algorithm
func (r *RateLimiter) checkRedisLimit(ctx context.Context, key string, rate float64, burst int) (bool, error) {
	// Create keys for the token count and last update time
	tokenKey := fmt.Sprintf("ratelimit:%s:tokens", key)
	timeKey := fmt.Sprintf("ratelimit:%s:ts", key)

	now := time.Now().UnixMilli()
	pipe := r.redis.Pipeline()

	// Get current tokens and last update time
	tokensCmd := pipe.Get(ctx, tokenKey)
	lastUpdateCmd := pipe.Get(ctx, timeKey)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return false, err
	}

	// Get current token count or set to burst if key doesn't exist
	tokens, _ := tokensCmd.Float64()
	lastUpdate, _ := lastUpdateCmd.Int64()
	if err == redis.Nil {
		tokens = float64(burst)
		lastUpdate = now
	}

	// Calculate tokens to add based on time passed
	timePassed := float64(now-lastUpdate) / 1000.0 // Convert to seconds
	tokens = math.Min(float64(burst), tokens+(timePassed*rate))

	// Try to consume a token
	if tokens < 1 {
		return false, nil
	}

	// Update token count and timestamp
	pipe = r.redis.Pipeline()
	pipe.Set(ctx, tokenKey, tokens-1, time.Second)
	pipe.Set(ctx, timeKey, now, time.Second)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}
