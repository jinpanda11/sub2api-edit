package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"

	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

const (
	// rankingRedisKeyPrefix 今日消费排行 Redis Sorted Set 前缀（+ YYYY-MM-DD）。
	rankingRedisKeyPrefix = "rank:consumption:"
	// rankingWarmKeySuffix 冷启动预热标记：仅当该标记存在时 sorted set 才可视为权威快照。
	rankingWarmKeySuffix = ":warm"
	// rankingKeyTTL 排行键保留 48h，避免历史日期键堆积。
	rankingKeyTTL = 48 * time.Hour
	// rankingCacheTTL 排行响应内存缓存时长。
	rankingCacheTTL = 60 * time.Second
	// rankingTopN 榜单展示数量。
	rankingTopN = 20
)

// rankingRedisKey 返回某上海自然日的排行键。
func rankingRedisKey(date string) string {
	return rankingRedisKeyPrefix + date
}

func rankingWarmKey(date string) string {
	return rankingRedisKey(date) + rankingWarmKeySuffix
}

// ConsumptionRankingEntry 榜单条目。
type ConsumptionRankingEntry struct {
	Rank        int     `json:"rank"`
	MaskedEmail string  `json:"masked_email"`
	Amount      float64 `json:"amount"`
	// TotalTokens 当日累计消耗 token 量（输入+输出+缓存+图片 token 之和）。
	TotalTokens int64 `json:"total_tokens"`
}

// ConsumptionRankingResponse 今日消费排行榜响应。
type ConsumptionRankingResponse struct {
	Date      string                    `json:"date"`
	Timezone  string                    `json:"timezone"`
	Enabled   bool                      `json:"enabled"`
	List      []ConsumptionRankingEntry `json:"list"`
	UpdatedAt string                    `json:"updated_at"`
}

// rankingCacheEntry 内存缓存条目。
type rankingCacheEntry struct {
	date     string
	resp     *ConsumptionRankingResponse
	cachedAt time.Time
}

// ConsumptionRankingService 每日消费排行榜服务。
//
// 数据源：usage_logs 中 billing_type = balance（余额扣费）且 actual_cost > 0 的
// 成功计费记录，按 Asia/Shanghai 自然日聚合。
// 读路径：Redis Sorted Set（消费发生时异步 ZINCRBY）→ 冷启动/丢失时从 DB 全量
// 重建 → Redis 不可用时直接查库；最终结果叠加 60s 内存缓存。
type ConsumptionRankingService struct {
	entClient   *dbent.Client
	rdb         *redis.Client
	settingRepo SettingRepository
	mu          sync.Mutex
	cache       *rankingCacheEntry
}

// NewConsumptionRankingService 创建排行榜服务。
func NewConsumptionRankingService(entClient *dbent.Client, rdb *redis.Client, settingRepo SettingRepository) *ConsumptionRankingService {
	return &ConsumptionRankingService{
		entClient:   entClient,
		rdb:         rdb,
		settingRepo: settingRepo,
	}
}

// enabled 返回排行榜开关状态（默认启用）。
func (s *ConsumptionRankingService) enabled(ctx context.Context) bool {
	if s.settingRepo == nil {
		return true
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyConsumptionRankingEnabled)
	if err != nil {
		return true
	}
	return raw != "false"
}

// IncrementAsync 消费发生后异步累加今日排行（best-effort，失败不影响计费主流程）。
// 仅由计费路径在「余额扣费成功」时调用。
func (s *ConsumptionRankingService) IncrementAsync(userID int64, cost float64) {
	if s == nil || s.rdb == nil || userID <= 0 || cost <= 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		date := shanghaiDateStr(time.Now())
		key := rankingRedisKey(date)
		if err := s.rdb.ZIncrBy(ctx, key, cost, strconv.FormatInt(userID, 10)).Err(); err != nil {
			logger.LegacyPrintf("service.ranking", "[Ranking] ZIncrBy failed: user_id=%d cost=%f err=%v", userID, cost, err)
			return
		}
		_ = s.rdb.Expire(ctx, key, rankingKeyTTL).Err()
	}()
}

// GetToday 返回今日 Top 20 消费排行（公开接口）。排行榜关闭时返回 enabled=false 空榜。
func (s *ConsumptionRankingService) GetToday(ctx context.Context) (*ConsumptionRankingResponse, error) {
	date := shanghaiDateStr(time.Now())

	if !s.enabled(ctx) {
		return &ConsumptionRankingResponse{
			Date:      date,
			Timezone:  "Asia/Shanghai",
			Enabled:   false,
			List:      []ConsumptionRankingEntry{},
			UpdatedAt: time.Now().In(shanghaiLoc).Format(time.RFC3339),
		}, nil
	}

	if resp := s.cachedResponse(date); resp != nil {
		return resp, nil
	}

	// Redis 路径：预热完成则直接 ZREVRANGE，冷启动则从 DB 重建。
	if s.rdb != nil {
		resp, err := s.getFromRedis(ctx, date)
		if err == nil {
			s.storeCache(date, resp)
			return resp, nil
		}
		logger.LegacyPrintf("service.ranking", "[Ranking] Redis read failed, falling back to DB: %v", err)
	}

	resp, err := s.getFromDB(ctx, date)
	if err != nil {
		return nil, err
	}
	s.storeCache(date, resp)
	return resp, nil
}

// getFromRedis 从 Redis 读取排行；未预热时从 DB 全量重建并预热。
func (s *ConsumptionRankingService) getFromRedis(ctx context.Context, date string) (*ConsumptionRankingResponse, error) {
	warm, err := s.rdb.Exists(ctx, rankingWarmKey(date)).Result()
	if err != nil {
		return nil, err
	}
	if warm == 0 {
		return s.warmFromDB(ctx, date)
	}

	pairs, err := s.rdb.ZRevRangeWithScores(ctx, rankingRedisKey(date), 0, rankingTopN-1).Result()
	if err != nil {
		return nil, err
	}

	userIDs := make([]int64, 0, len(pairs))
	for _, p := range pairs {
		if id, err := strconv.ParseInt(p.Member.(string), 10, 64); err == nil {
			userIDs = append(userIDs, id)
		}
	}
	emails := s.fetchEmails(ctx, userIDs)
	tokenTotals := s.fetchTokenTotals(ctx, userIDs, date)

	list := make([]ConsumptionRankingEntry, 0, len(pairs))
	for i, p := range pairs {
		userID, _ := strconv.ParseInt(p.Member.(string), 10, 64)
		email := emails[userID]
		if email == "" {
			continue
		}
		list = append(list, ConsumptionRankingEntry{
			Rank:        i + 1,
			MaskedEmail: MaskRankingEmail(email),
			Amount:      round4(p.Score),
			TotalTokens: tokenTotals[userID],
		})
	}
	return &ConsumptionRankingResponse{
		Date:      date,
		Timezone:  "Asia/Shanghai",
		Enabled:   true,
		List:      list,
		UpdatedAt: time.Now().In(shanghaiLoc).Format(time.RFC3339),
	}, nil
}

// warmFromDB 冷启动：从 DB 聚合今日 Top 20 写入 Redis 并打预热标记。
func (s *ConsumptionRankingService) warmFromDB(ctx context.Context, date string) (*ConsumptionRankingResponse, error) {
	rows, err := s.aggregateToday(ctx, date)
	if err != nil {
		return nil, err
	}

	key := rankingRedisKey(date)
	if len(rows) > 0 {
		members := make([]redis.Z, 0, len(rows))
		for _, r := range rows {
			members = append(members, redis.Z{
				Score:  r.amount,
				Member: strconv.FormatInt(r.userID, 10),
			})
		}
		if err := s.rdb.ZAdd(ctx, key, members...).Err(); err != nil {
			return nil, err
		}
	}
	if err := s.rdb.Set(ctx, rankingWarmKey(date), "1", rankingKeyTTL).Err(); err != nil {
		return nil, err
	}
	_ = s.rdb.Expire(ctx, key, rankingKeyTTL).Err()

	return s.buildResponseFromRows(rows, date), nil
}

// getFromDB Redis 不可用时的兜底：直接聚合今日排行。
func (s *ConsumptionRankingService) getFromDB(ctx context.Context, date string) (*ConsumptionRankingResponse, error) {
	rows, err := s.aggregateToday(ctx, date)
	if err != nil {
		return nil, err
	}
	return s.buildResponseFromRows(rows, date), nil
}

func (s *ConsumptionRankingService) buildResponseFromRows(rows []rankingAggRow, date string) *ConsumptionRankingResponse {
	userIDs := make([]int64, 0, len(rows))
	for _, r := range rows {
		userIDs = append(userIDs, r.userID)
	}
	emails := s.fetchEmails(context.Background(), userIDs)

	list := make([]ConsumptionRankingEntry, 0, len(rows))
	for i, r := range rows {
		email := emails[r.userID]
		if email == "" {
			continue
		}
		list = append(list, ConsumptionRankingEntry{
			Rank:        i + 1,
			MaskedEmail: MaskRankingEmail(email),
			Amount:      round4(r.amount),
			TotalTokens: r.totalTokens,
		})
	}
	return &ConsumptionRankingResponse{
		Date:      date,
		Timezone:  "Asia/Shanghai",
		Enabled:   true,
		List:      list,
		UpdatedAt: time.Now().In(shanghaiLoc).Format(time.RFC3339),
	}
}

// rankingAggRow DB 聚合结果行。
type rankingAggRow struct {
	userID      int64
	amount      float64
	totalTokens int64
}

// aggregateToday 按上海自然日聚合余额扣费（billing_type=0）的实际消费 Top 20。
// 同时聚合当日累计 token 量（输入+输出+缓存+图片 token）。
func (s *ConsumptionRankingService) aggregateToday(ctx context.Context, date string) ([]rankingAggRow, error) {
	start, err := time.ParseInLocation("2006-01-02", date, shanghaiLoc)
	if err != nil {
		return nil, fmt.Errorf("parse ranking date: %w", err)
	}
	end := start.Add(24 * time.Hour)

	rows, err := s.entClient.QueryContext(ctx, `
		SELECT user_id,
		       SUM(actual_cost)::double precision AS amount,
		       SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens +
		           image_input_tokens + image_output_tokens)::bigint AS total_tokens
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
		  AND billing_type = 0
		  AND actual_cost > 0
		GROUP BY user_id
		ORDER BY amount DESC, user_id ASC
		LIMIT $3`, start, end, rankingTopN)
	if err != nil {
		return nil, fmt.Errorf("aggregate consumption ranking: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]rankingAggRow, 0, rankingTopN)
	for rows.Next() {
		var r rankingAggRow
		if err := rows.Scan(&r.userID, &r.amount, &r.totalTokens); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// fetchTokenTotals 按 user 批量查询指定上海自然日的累计 token 量（Redis 实时路径补查用）。
func (s *ConsumptionRankingService) fetchTokenTotals(ctx context.Context, userIDs []int64, date string) map[int64]int64 {
	out := make(map[int64]int64, len(userIDs))
	if len(userIDs) == 0 || s.entClient == nil {
		return out
	}
	start, err := time.ParseInLocation("2006-01-02", date, shanghaiLoc)
	if err != nil {
		return out
	}
	end := start.Add(24 * time.Hour)

	rows, err := s.entClient.QueryContext(ctx, `
		SELECT user_id,
		       SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens +
		           image_input_tokens + image_output_tokens)::bigint AS total_tokens
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
		  AND billing_type = 0
		  AND user_id = ANY($3)
		GROUP BY user_id`, start, end, pq.Array(userIDs))
	if err != nil {
		logger.LegacyPrintf("service.ranking", "[Ranking] fetch token totals failed: %v", err)
		return out
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var userID int64
		var total int64
		if err := rows.Scan(&userID, &total); err != nil {
			logger.LegacyPrintf("service.ranking", "[Ranking] scan token totals failed: %v", err)
			return out
		}
		out[userID] = total
	}
	return out
}

// fetchEmails 批量获取用户邮箱（缺失/已删除用户返回空串）。
func (s *ConsumptionRankingService) fetchEmails(ctx context.Context, userIDs []int64) map[int64]string {
	out := make(map[int64]string, len(userIDs))
	if len(userIDs) == 0 || s.entClient == nil {
		return out
	}
	users, err := s.entClient.User.Query().
		Where(dbuser.IDIn(userIDs...), dbuser.DeletedAtIsNil()).
		Select(dbuser.FieldID, dbuser.FieldEmail).
		All(ctx)
	if err != nil {
		logger.LegacyPrintf("service.ranking", "[Ranking] fetch emails failed: %v", err)
		return out
	}
	for _, u := range users {
		out[u.ID] = u.Email
	}
	return out
}

func (s *ConsumptionRankingService) cachedResponse(date string) *ConsumptionRankingResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cache != nil && s.cache.date == date && time.Since(s.cache.cachedAt) < rankingCacheTTL {
		return s.cache.resp
	}
	return nil
}

func (s *ConsumptionRankingService) storeCache(date string, resp *ConsumptionRankingResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = &rankingCacheEntry{date: date, resp: resp, cachedAt: time.Now()}
}

// MaskRankingEmail 按排行榜规范脱敏邮箱：本地部分前 2 位 + 自适应 * 填充 + 后 2 位，域名保留。
// 示例：ab123456cd@qq.com → ab****cd@qq.com；testuser@example.com → te****er@example.com。
// 本地部分过短（<=4）时退化为 首字符 + ***。
func MaskRankingEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return "***@***"
	}
	local, domain := email[:at], email[at+1:]
	if len(local) <= 4 {
		return local[:1] + "***@" + domain
	}
	return local[:2] + strings.Repeat("*", len(local)-4) + local[len(local)-2:] + "@" + domain
}

// round4 金额保留 4 位小数（对齐文档「精确到小数点后 4 位」）。
func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
