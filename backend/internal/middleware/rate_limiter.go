package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gocommerce-backend/internal/config"
	"gocommerce-backend/internal/logger"
	"gocommerce-backend/internal/utils"
)

// RateLimiter returns a Gin middleware that enforces distributed rate limiting via Redis
func RateLimiter(rdb *redis.Client, cfg *config.Config) gin.HandlerFunc {
	limit := cfg.Redis.RateLimitReqPerMinute
	if limit <= 0 {
		limit = 120 // Fallback default
	}

	windowSecs := cfg.Redis.RateLimitWindowSeconds
	if windowSecs <= 0 {
		windowSecs = 60
	}

	return func(c *gin.Context) {
		// If Redis is not initialized, fail open and proceed
		if rdb == nil {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		clientID := c.ClientIP()
		if clientID == "" {
			clientID = "unknown"
		}

		// Use time window bucket (e.g. 60-second slice)
		nowUnix := time.Now().Unix()
		currentWindow := nowUnix / int64(windowSecs)
		resetSecs := int64(windowSecs) - (nowUnix % int64(windowSecs))
		if resetSecs <= 0 {
			resetSecs = int64(windowSecs)
		}

		key := fmt.Sprintf("ratelimit:%s:%d", clientID, currentWindow)

		// Execute atomic Redis INCR & EXPIRE in a pipeline
		pipe := rdb.Pipeline()
		incrCmd := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, time.Duration(windowSecs*2)*time.Second)

		_, err := pipe.Exec(ctx)
		if err != nil {
			// Fail open on Redis error so backend remains operational
			logger.Warn(ctx, "middleware", fmt.Sprintf("Redis rate limiter error for IP %s (failing open)", clientID), err)
			c.Next()
			return
		}

		currentCount := incrCmd.Val()
		remaining := int64(limit) - currentCount
		if remaining < 0 {
			remaining = 0
		}

		// Set standard RateLimit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetSecs, 10))

		// If current request exceeds allowed threshold, reject with 429 Too Many Requests
		if currentCount > int64(limit) {
			c.Header("Retry-After", strconv.FormatInt(resetSecs, 10))

			errDesc := fmt.Sprintf("Rate limit exceeded for IP %s (%d/%d reqs in %ds)", clientID, currentCount, limit, windowSecs)
			logger.Warn(ctx, "middleware", errDesc, nil)

			utils.Error(
				c,
				http.StatusTooManyRequests,
				fmt.Sprintf("Rate limit exceeded (%d req/%ds). Please wait %d seconds.", limit, windowSecs, resetSecs),
				fmt.Sprintf("limit_%d_exceeded", limit),
			)
			c.Abort()
			return
		}

		c.Next()
	}
}
