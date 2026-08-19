package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/redis/go-redis/v9"
)

// lotteryDrawLimiter 实现 service.LotteryDrawLimiter（Redis SetNX 1 次/秒限流）。
type lotteryDrawLimiter struct {
	rdb *redis.Client
}

// NewLotteryDrawLimiter 创建抽奖防刷限流器。
func NewLotteryDrawLimiter(rdb *redis.Client) service.LotteryDrawLimiter {
	return &lotteryDrawLimiter{rdb: rdb}
}

func (l *lotteryDrawLimiter) TryAcquireDrawToken(ctx context.Context, userID int64) bool {
	if l == nil || l.rdb == nil {
		return true // Redis 不可用时降级为不限制
	}
	ok, err := l.rdb.SetNX(ctx, fmt.Sprintf("lottery:draw:%d", userID), 1, time.Second).Result()
	if err != nil {
		return true
	}
	return ok
}
