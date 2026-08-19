package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/redis/go-redis/v9"
)

const (
	// consumptionRankingKeyPrefix 今日消费排行 Redis Sorted Set 前缀（+ YYYY-MM-DD）。
	consumptionRankingKeyPrefix = "rank:consumption:"
	// consumptionRankingWarmSuffix 冷启动预热标记：仅当该标记存在时 sorted set 才可视为权威快照。
	consumptionRankingWarmSuffix = ":warm"
	// consumptionRankingKeyTTL 排行键保留 48h，避免历史日期键堆积。
	consumptionRankingKeyTTL = 48 * time.Hour
)

func consumptionRankingKey(date string) string {
	return consumptionRankingKeyPrefix + date
}

func consumptionRankingWarmKey(date string) string {
	return consumptionRankingKey(date) + consumptionRankingWarmSuffix
}

// consumptionRankingCache 实现 service.ConsumptionRankingCache（Redis Sorted Set）。
type consumptionRankingCache struct {
	rdb *redis.Client
}

// NewConsumptionRankingCache 创建排行榜缓存。
func NewConsumptionRankingCache(rdb *redis.Client) service.ConsumptionRankingCache {
	return &consumptionRankingCache{rdb: rdb}
}

func (c *consumptionRankingCache) IncrementToday(ctx context.Context, date string, userID int64, cost float64) error {
	key := consumptionRankingKey(date)
	if err := c.rdb.ZIncrBy(ctx, key, cost, strconv.FormatInt(userID, 10)).Err(); err != nil {
		return err
	}
	return c.rdb.Expire(ctx, key, consumptionRankingKeyTTL).Err()
}

func (c *consumptionRankingCache) IsWarm(ctx context.Context, date string) (bool, error) {
	n, err := c.rdb.Exists(ctx, consumptionRankingWarmKey(date)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (c *consumptionRankingCache) TopUsers(ctx context.Context, date string, limit int) ([]service.ConsumptionRankingScore, error) {
	pairs, err := c.rdb.ZRevRangeWithScores(ctx, consumptionRankingKey(date), 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]service.ConsumptionRankingScore, 0, len(pairs))
	for _, p := range pairs {
		member, ok := p.Member.(string)
		if !ok {
			continue
		}
		userID, err := strconv.ParseInt(member, 10, 64)
		if err != nil {
			continue
		}
		out = append(out, service.ConsumptionRankingScore{UserID: userID, Score: p.Score})
	}
	return out, nil
}

func (c *consumptionRankingCache) MarkWarm(ctx context.Context, date string) error {
	return c.rdb.Set(ctx, consumptionRankingWarmKey(date), "1", consumptionRankingKeyTTL).Err()
}

func (c *consumptionRankingCache) WarmSet(ctx context.Context, date string, scores []service.ConsumptionRankingScore) error {
	key := consumptionRankingKey(date)
	if len(scores) == 0 {
		return c.rdb.Expire(ctx, key, consumptionRankingKeyTTL).Err()
	}
	members := make([]redis.Z, 0, len(scores))
	for _, sc := range scores {
		members = append(members, redis.Z{
			Score:  sc.Score,
			Member: strconv.FormatInt(sc.UserID, 10),
		})
	}
	if err := c.rdb.ZAdd(ctx, key, members...).Err(); err != nil {
		return err
	}
	return c.rdb.Expire(ctx, key, consumptionRankingKeyTTL).Err()
}
