package limiter

import (
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"time"
)

func NewGinLimiter(period time.Duration, limit int64) gin.HandlerFunc {
	rate := limiter.Rate{
		Period: period * time.Hour,
		Limit:  limit,
	}
	store := memory.NewStore()
	return mgin.NewMiddleware(limiter.New(store, rate))
}
