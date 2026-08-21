package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	// rankingTopN 榜单展示数量。
	rankingTopN = 20
)

// ConsumptionRankingEntry 榜单条目。
type ConsumptionRankingEntry struct {
	Rank        int     `json:"rank"`
	MaskedEmail string  `json:"masked_email"`
	Amount      float64 `json:"amount"`
	// TotalTokens 当日累计消耗 token 量（输入、输出和缓存桶之和）。
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

// ConsumptionRankingScore 排行榜分数条目（cache 层返回）。
type ConsumptionRankingScore struct {
	UserID int64
	Score  float64
}

// ConsumptionRankingCache 保留 Redis 实现的接口兼容性；公开读取不再依赖该缓存。
type ConsumptionRankingCache interface {
	IncrementToday(ctx context.Context, date string, userID int64, cost float64) error
	IsWarm(ctx context.Context, date string) (bool, error)
	TopUsers(ctx context.Context, date string, limit int) ([]ConsumptionRankingScore, error)
	MarkWarm(ctx context.Context, date string) error
	WarmSet(ctx context.Context, date string, scores []ConsumptionRankingScore) error
}

// ConsumptionRankingService 每日消费排行榜服务。
//
// 数据源：usage_logs 中 billing_type = balance（余额扣费）且 actual_cost > 0 的
// 成功计费记录，按 Asia/Shanghai 自然日聚合。公开读取始终以数据库为准。
type ConsumptionRankingService struct {
	entClient   *dbent.Client
	settingRepo SettingRepository
}

// NewConsumptionRankingService 创建排行榜服务。
func NewConsumptionRankingService(entClient *dbent.Client, _ ConsumptionRankingCache, settingRepo SettingRepository) *ConsumptionRankingService {
	return &ConsumptionRankingService{
		entClient:   entClient,
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

// GetToday 返回今日 Top 20 消费排行（公开接口）。排行榜关闭时返回 enabled=false 空榜。
// 每次请求直接读取 usage_logs，避免进程内缓存和 Redis 增量投影造成陈旧或不一致结果。
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

	rows, err := s.aggregateToday(ctx, date)
	if err != nil {
		return nil, err
	}
	return s.buildResponseFromRows(ctx, rows, date), nil
}

// buildResponseFromRows 构造公开榜单响应。
func (s *ConsumptionRankingService) buildResponseFromRows(ctx context.Context, rows []rankingAggRow, date string) *ConsumptionRankingResponse {
	userIDs := make([]int64, 0, len(rows))
	for _, r := range rows {
		userIDs = append(userIDs, r.userID)
	}
	emails := s.fetchEmails(ctx, userIDs)

	list := make([]ConsumptionRankingEntry, 0, len(rows))
	for _, r := range rows {
		email := emails[r.userID]
		if email == "" {
			continue
		}
		list = append(list, ConsumptionRankingEntry{
			Rank:        len(list) + 1,
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
// Token 口径与管理员用量排行一致，只统计基础输入、输出和缓存桶。
func (s *ConsumptionRankingService) aggregateToday(ctx context.Context, date string) ([]rankingAggRow, error) {
	start, err := time.ParseInLocation("2006-01-02", date, shanghaiLoc)
	if err != nil {
		return nil, fmt.Errorf("parse ranking date: %w", err)
	}
	end := start.Add(24 * time.Hour)

	rows, err := s.entClient.QueryContext(ctx, `
		SELECT user_id,
		       COALESCE(SUM(actual_cost), 0) AS amount,
		       COALESCE(SUM(CAST(input_tokens AS BIGINT) + CAST(output_tokens AS BIGINT) +
		           CAST(cache_creation_tokens AS BIGINT) + CAST(cache_read_tokens AS BIGINT)), 0) AS total_tokens
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
