package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	defaultLotteryMinAmount        = 0.10
	defaultLotteryMaxAmount        = 2.00
	defaultLotteryRechargeUnit     = 10.0
	defaultLotteryRechargeDailyMax = 5
	defaultLotteryLoginReward      = 1
)

var (
	ErrLotteryChanceNotFound = errors.NotFound("LOTTERY_CHANCE_NOT_FOUND", "lottery chance account not found")
	ErrLotteryDisabled       = errors.Forbidden("LOTTERY_DISABLED", "lottery is currently disabled")
	ErrLotteryNoChances      = errors.BadRequest("LOTTERY_NO_CHANCES", "no lottery chances available")
	ErrLotteryTooFrequent    = errors.TooManyRequests("LOTTERY_RATE_LIMITED", "drawing too frequently, please try again later")
)

// shanghaiLoc 固定 +08:00 时区。中国无夏令时，FixedZone 无需 tzdata，且行为确定。
var shanghaiLoc = time.FixedZone("Asia/Shanghai", 8*3600)

// shanghaiDateStr 返回上海时区的自然日 YYYY-MM-DD。
func shanghaiDateStr(t time.Time) string {
	return t.In(shanghaiLoc).Format("2006-01-02")
}

// LotteryChance 用户抽奖次数账户。
type LotteryChance struct {
	ID                 int64
	UserID             int64
	AvailableCount     int
	TodayRechargeCount int
	RechargeDate       *string
	LastLoginDate      *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// LotteryRecord 中奖明细。
type LotteryRecord struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Amount       float64   `json:"amount"`
	BalanceAfter float64   `json:"balance_after"`
	Source       string    `json:"source"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// LotteryChanceRepository 抽奖次数账户存储。
type LotteryChanceRepository interface {
	GetOrCreate(ctx context.Context, userID int64) (*LotteryChance, error)
	Get(ctx context.Context, userID int64) (*LotteryChance, error)
	// ResetDailyCountIfExpired 当可用次数所属上海自然日早于 today 时清零并标记 today。
	// 用于实现“未使用次数不跨天保留”。
	ResetDailyCountIfExpired(ctx context.Context, userID int64, today string) error
	// GrantLoginReward 原子发放登录奖励，仅当 last_login_date < today 时生效。
	GrantLoginReward(ctx context.Context, userID int64, reward int, today string) (bool, error)
	// GrantRechargeReward 原子发放充值/兑换码奖励，返回实际发放次数（受每日上限约束）。
	GrantRechargeReward(ctx context.Context, userID int64, earned int, today string, dailyMax int) (int, error)
	// ConsumeChance 原子消耗 1 次可用次数。
	ConsumeChance(ctx context.Context, userID int64) (bool, error)
}

// LotteryRecordRepository 中奖记录存储。
type LotteryRecordRepository interface {
	// Create 写入中奖记录并返回记录 ID。
	Create(ctx context.Context, record *LotteryRecord) (int64, error)
	// UpdateBalanceAfter 回填抽奖后的真实余额（事务内读不到未提交余额，需提交后回填）。
	UpdateBalanceAfter(ctx context.Context, id int64, balanceAfter float64) error
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]LotteryRecord, error)
	CountByUser(ctx context.Context, userID int64) (int64, error)
	// DailyAggregate 按上海自然日聚合 [from, to) 区间内的抽奖记录，并返回区间内去重参与人数。
	DailyAggregate(ctx context.Context, from, to time.Time) ([]LotteryDailyAggregate, int64, error)
}

// LotteryConfig 抽奖规则配置（来自 settings 表，缺失时使用默认值）。
type LotteryConfig struct {
	Enabled          bool
	MinAmount        float64
	MaxAmount        float64
	RechargeUnit     float64
	RechargeDailyMax int
	LoginReward      int
}

// LotteryStatus 抽奖状态（GET /api/v1/lottery/status）。
type LotteryStatus struct {
	Enabled            bool    `json:"enabled"`
	AvailableCount     int     `json:"available_count"`
	TodayRechargeCount int     `json:"today_recharge_count"`
	TodayRechargeMax   int     `json:"today_recharge_max"`
	LoginRewardedToday bool    `json:"login_rewarded_today"`
	CurrentBalance     float64 `json:"current_balance"`
	MinPrize           float64 `json:"min_prize"`
	MaxPrize           float64 `json:"max_prize"`
	RechargeUnit       float64 `json:"recharge_unit"`
	LoginReward        int     `json:"login_reward"`
	Today              string  `json:"today"`
	Timezone           string  `json:"timezone"`
}

// DrawResult 抽奖结果（POST /api/v1/lottery/draw）。
type DrawResult struct {
	PrizeAmount    float64 `json:"prize_amount"`
	NewBalance     float64 `json:"new_balance"`
	RemainingCount int     `json:"remaining_count"`
}

// LotteryDailyAggregate 某上海自然日内的抽奖聚合结果。
type LotteryDailyAggregate struct {
	Date         string  `json:"date"`
	Draws        int64   `json:"draws"`
	Participants int64   `json:"participants"`
	TotalAmount  float64 `json:"total_amount"`
}

// LotteryDailyStats 管理端每日抽奖统计（GET /api/v1/admin/lottery/stats）。
type LotteryDailyStats struct {
	StartDate string                  `json:"start_date"`
	EndDate   string                  `json:"end_date"`
	Summary   LotteryDailySummary     `json:"summary"`
	Daily     []LotteryDailyAggregate `json:"daily"`
}

// LotteryDailySummary 日期区间内的统计汇总。
type LotteryDailySummary struct {
	TotalDraws        int64   `json:"total_draws"`
	TotalParticipants int64   `json:"total_participants"`
	TotalAmount       float64 `json:"total_amount"`
	AvgAmount         float64 `json:"avg_amount"`
}

// LotteryDrawLimiter 抽奖接口防刷限流（1 次/秒），由 repository 层基于 Redis 实现。
type LotteryDrawLimiter interface {
	TryAcquireDrawToken(ctx context.Context, userID int64) bool
}

// LotteryService 每日盲盒抽奖服务。
type LotteryService struct {
	entClient           *dbent.Client
	userRepo            UserRepository
	chanceRepo          LotteryChanceRepository
	recordRepo          LotteryRecordRepository
	settingRepo         SettingRepository
	billingCacheService *BillingCacheService
	drawLimiter         LotteryDrawLimiter
}

// NewLotteryService 创建 LotteryService。
func NewLotteryService(
	entClient *dbent.Client,
	userRepo UserRepository,
	chanceRepo LotteryChanceRepository,
	recordRepo LotteryRecordRepository,
	settingRepo SettingRepository,
	billingCacheService *BillingCacheService,
	drawLimiter LotteryDrawLimiter,
) *LotteryService {
	return &LotteryService{
		entClient:           entClient,
		userRepo:            userRepo,
		chanceRepo:          chanceRepo,
		recordRepo:          recordRepo,
		settingRepo:         settingRepo,
		billingCacheService: billingCacheService,
		drawLimiter:         drawLimiter,
	}
}

// GetConfig 读取抽奖配置，settings 缺失或解析失败时回退默认值。
func (s *LotteryService) GetConfig(ctx context.Context) LotteryConfig {
	cfg := LotteryConfig{
		Enabled:          true,
		MinAmount:        defaultLotteryMinAmount,
		MaxAmount:        defaultLotteryMaxAmount,
		RechargeUnit:     defaultLotteryRechargeUnit,
		RechargeDailyMax: defaultLotteryRechargeDailyMax,
		LoginReward:      defaultLotteryLoginReward,
	}
	if s.settingRepo == nil {
		return cfg
	}
	vals, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyLotteryEnabled,
		SettingKeyLotteryMinAmount,
		SettingKeyLotteryMaxAmount,
		SettingKeyLotteryRechargeUnit,
		SettingKeyLotteryRechargeDailyMax,
		SettingKeyLotteryLoginReward,
	})
	if err != nil {
		return cfg
	}
	if v, ok := vals[SettingKeyLotteryEnabled]; ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Enabled = b
		}
	}
	if v, ok := vals[SettingKeyLotteryMinAmount]; ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			cfg.MinAmount = f
		}
	}
	if v, ok := vals[SettingKeyLotteryMaxAmount]; ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			cfg.MaxAmount = f
		}
	}
	if v, ok := vals[SettingKeyLotteryRechargeUnit]; ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			cfg.RechargeUnit = f
		}
	}
	if v, ok := vals[SettingKeyLotteryRechargeDailyMax]; ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.RechargeDailyMax = n
		}
	}
	if v, ok := vals[SettingKeyLotteryLoginReward]; ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.LoginReward = n
		}
	}
	return cfg
}

// GrantLoginReward 发放登录奖励（每日一次，幂等）。
// 由登录成功钩子调用；Status/Draw 也会做惰性兜底发放。
func (s *LotteryService) GrantLoginReward(ctx context.Context, userID int64, reward int) error {
	if reward <= 0 {
		return nil
	}
	if _, err := s.chanceRepo.GetOrCreate(ctx, userID); err != nil {
		return err
	}
	today := shanghaiDateStr(time.Now())
	// 未使用次数不跨天保留：进入新的一天先清零旧次数。
	if err := s.chanceRepo.ResetDailyCountIfExpired(ctx, userID, today); err != nil {
		return err
	}
	_, err := s.chanceRepo.GrantLoginReward(ctx, userID, reward, today)
	return err
}

// GrantRechargeReward 按充值/兑换码入账金额发放抽奖次数（每 recharge_unit 余额 +1，当日上限）。
// 由兑换/充值入账流程在同一事务内调用；best-effort，失败不影响入账结果。
func (s *LotteryService) GrantRechargeReward(ctx context.Context, userID int64, amount float64) error {
	if amount <= 0 {
		return nil
	}
	cfg := s.GetConfig(ctx)
	if !cfg.Enabled || cfg.RechargeUnit <= 0 || cfg.RechargeDailyMax <= 0 {
		return nil
	}
	earned := int(math.Floor(amount / cfg.RechargeUnit))
	if earned <= 0 {
		return nil
	}
	if _, err := s.chanceRepo.GetOrCreate(ctx, userID); err != nil {
		return err
	}
	today := shanghaiDateStr(time.Now())
	// 未使用次数不跨天保留：进入新的一天先清零旧次数，再计入今日充值所得。
	if err := s.chanceRepo.ResetDailyCountIfExpired(ctx, userID, today); err != nil {
		return err
	}
	_, err := s.chanceRepo.GrantRechargeReward(ctx, userID, earned, today, cfg.RechargeDailyMax)
	return err
}

// Status 返回用户抽奖状态（含惰性登录奖励兜底发放）。
func (s *LotteryService) Status(ctx context.Context, userID int64) (*LotteryStatus, error) {
	cfg := s.GetConfig(ctx)
	today := shanghaiDateStr(time.Now())

	// 未使用次数不跨天保留：进入新的一天先清零旧次数。
	if _, err := s.chanceRepo.GetOrCreate(ctx, userID); err != nil {
		return nil, err
	}
	if err := s.chanceRepo.ResetDailyCountIfExpired(ctx, userID, today); err != nil {
		return nil, err
	}

	// 惰性兜底：任何登录路径（含 passkey 等未走登录钩子的路径）只要当天登录过就补发奖励。
	_ = s.ensureLoginReward(ctx, userID, cfg, today)

	chance, err := s.chanceRepo.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	loginRewardedToday := chance.LastLoginDate != nil && *chance.LastLoginDate == today
	return &LotteryStatus{
		Enabled:            cfg.Enabled,
		AvailableCount:     chance.AvailableCount,
		TodayRechargeCount: chance.TodayRechargeCount,
		TodayRechargeMax:   cfg.RechargeDailyMax,
		LoginRewardedToday: loginRewardedToday,
		CurrentBalance:     user.Balance,
		MinPrize:           cfg.MinAmount,
		MaxPrize:           cfg.MaxAmount,
		RechargeUnit:       cfg.RechargeUnit,
		LoginReward:        cfg.LoginReward,
		Today:              today,
		Timezone:           "Asia/Shanghai",
	}, nil
}

// Draw 执行一次抽奖：扣次数 → 生成随机金额 → 加余额 → 写记录，同一事务原子提交。
func (s *LotteryService) Draw(ctx context.Context, userID int64) (*DrawResult, error) {
	cfg := s.GetConfig(ctx)
	if !cfg.Enabled {
		return nil, ErrLotteryDisabled
	}
	// 防刷：同一用户 1 次/秒（Redis SetNX，Redis 不可用时降级为不限制）。
	if !s.acquireDrawToken(ctx, userID) {
		return nil, ErrLotteryTooFrequent
	}
	// 未使用次数不跨天保留：进入新的一天先清零旧次数，再发放今日登录奖励。
	if _, err := s.chanceRepo.GetOrCreate(ctx, userID); err != nil {
		return nil, err
	}
	today := shanghaiDateStr(time.Now())
	if err := s.chanceRepo.ResetDailyCountIfExpired(ctx, userID, today); err != nil {
		return nil, err
	}
	// 惰性兜底发放登录奖励。
	_ = s.ensureLoginReward(ctx, userID, cfg, today)

	if _, err := s.chanceRepo.GetOrCreate(ctx, userID); err != nil {
		return nil, err
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	consumed, err := s.chanceRepo.ConsumeChance(txCtx, userID)
	if err != nil {
		return nil, fmt.Errorf("consume chance: %w", err)
	}
	if !consumed {
		return nil, ErrLotteryNoChances
	}

	prize := uniformPrize(cfg.MinAmount, cfg.MaxAmount)
	if err := s.userRepo.UpdateBalance(txCtx, userID, prize); err != nil {
		return nil, fmt.Errorf("update user balance: %w", err)
	}
	// 注意：事务内 userRepo.GetByID 走非事务 client，读不到未提交的余额，
	// 因此记录先写占位（balance_after=0），提交后统一回填真实余额。
	recordID, err := s.recordRepo.Create(txCtx, &LotteryRecord{
		UserID: userID,
		Amount: prize,
		Source: "draw",
	})
	if err != nil {
		return nil, fmt.Errorf("create lottery record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	s.invalidateBalanceCache(ctx, userID)

	// 提交后读取真实余额（含本事务的入账）。
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user after draw: %w", err)
	}
	if err := s.recordRepo.UpdateBalanceAfter(ctx, recordID, user.Balance); err != nil {
		logger.LegacyPrintf("service.lottery", "[Lottery] Failed to backfill balance_after: record_id=%d err=%v", recordID, err)
	}

	remaining := 0
	if chance, err := s.chanceRepo.Get(ctx, userID); err == nil {
		remaining = chance.AvailableCount
	}
	return &DrawResult{
		PrizeAmount:    prize,
		NewBalance:     user.Balance,
		RemainingCount: remaining,
	}, nil
}

// Records 分页返回用户中奖记录。
func (s *LotteryService) Records(ctx context.Context, userID int64, page, pageSize int) ([]LotteryRecord, *pagination.PaginationResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	total, err := s.recordRepo.CountByUser(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	records, err := s.recordRepo.ListByUser(ctx, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, nil, err
	}
	pages := int(math.Ceil(float64(total) / float64(pageSize)))
	if pages < 1 {
		pages = 1
	}
	return records, &pagination.PaginationResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    pages,
	}, nil
}

// AdminDailyStats 返回 [startDate, endDate] 区间（上海自然日）内的每日抽奖统计汇总。
// 日期为空时默认取最近 30 天；区间含零值日期补齐，便于前端按天展表。
func (s *LotteryService) AdminDailyStats(ctx context.Context, startDate, endDate string) (*LotteryDailyStats, error) {
	start, end, err := resolveLotteryStatsWindow(startDate, endDate)
	if err != nil {
		return nil, err
	}
	if s.recordRepo == nil {
		return nil, fmt.Errorf("lottery record repository not configured")
	}

	// end 已指向次日的 00:00（上海时区），聚合使用 [start, end) 半开区间。
	aggregates, rangeParticipants, err := s.recordRepo.DailyAggregate(ctx, start, end)
	if err != nil {
		return nil, err
	}
	byDate := make(map[string]LotteryDailyAggregate, len(aggregates))
	for _, a := range aggregates {
		byDate[a.Date] = a
	}

	daily := make([]LotteryDailyAggregate, 0, end.Sub(start)/(24*time.Hour))
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if agg, ok := byDate[dateStr]; ok {
			daily = append(daily, agg)
		} else {
			daily = append(daily, LotteryDailyAggregate{Date: dateStr})
		}
	}

	summary := LotteryDailySummary{TotalParticipants: rangeParticipants}
	for _, a := range aggregates {
		summary.TotalDraws += a.Draws
		summary.TotalAmount += a.TotalAmount
	}
	if summary.TotalDraws > 0 {
		summary.AvgAmount = math.Round(summary.TotalAmount/float64(summary.TotalDraws)*100) / 100
	}
	return &LotteryDailyStats{
		StartDate: start.Format("2006-01-02"),
		EndDate:   end.AddDate(0, 0, -1).Format("2006-01-02"),
		Summary:   summary,
		Daily:     daily,
	}, nil
}

// resolveLotteryStatsWindow 解析统计日期窗口，返回 [start, endOfEnd+1day) 上海时区边界。
// 传入合法 YYYY-MM-DD 则使用；为空时默认最近 30 天。
func resolveLotteryStatsWindow(startDate, endDate string) (time.Time, time.Time, error) {
	now := time.Now().In(shanghaiLoc)

	var start time.Time
	var end time.Time
	if startDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", startDate, shanghaiLoc)
		if err != nil {
			return time.Time{}, time.Time{}, errors.BadRequest("LOTTERY_STATS_INVALID_DATE", "invalid start_date, expected YYYY-MM-DD")
		}
		start = parsed
	}
	if endDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", endDate, shanghaiLoc)
		if err != nil {
			return time.Time{}, time.Time{}, errors.BadRequest("LOTTERY_STATS_INVALID_DATE", "invalid end_date, expected YYYY-MM-DD")
		}
		end = parsed
	}
	if startDate == "" {
		start = now.AddDate(0, 0, -29)
	}
	if endDate == "" {
		end = now
	}
	// start/end 归一化为当天的 00:00（上海时区）。
	start = shanghaiStartOfDay(start)
	end = shanghaiStartOfDay(end)
	// end 为闭区间日期，exclusiveEnd 为次日 00:00，保证聚合为 [start, exclusiveEnd) 半开区间。
	exclusiveEnd := end.AddDate(0, 0, 1)
	// 防御：非法倒序窗口时兜底为当天。
	if !start.Before(exclusiveEnd) {
		start = exclusiveEnd.AddDate(0, 0, -1)
	}
	return start, exclusiveEnd, nil
}

// shanghaiStartOfDay 返回给定时刻所在上海自然日的 00:00。
func shanghaiStartOfDay(t time.Time) time.Time {
	in := t.In(shanghaiLoc)
	return time.Date(in.Year(), in.Month(), in.Day(), 0, 0, 0, 0, shanghaiLoc)
}

// ensureLoginReward 惰性登录奖励：仅当用户当天（上海时区）成功登录过且奖励未发放时补发。
// best-effort：任何失败都不影响主流程。
func (s *LotteryService) ensureLoginReward(ctx context.Context, userID int64, cfg LotteryConfig, today string) error {
	if !cfg.Enabled || cfg.LoginReward <= 0 {
		return nil
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user.LastLoginAt == nil {
		return nil
	}
	if shanghaiDateStr(*user.LastLoginAt) != today {
		return nil
	}
	if _, err := s.chanceRepo.GetOrCreate(ctx, userID); err != nil {
		return nil
	}
	_, err = s.chanceRepo.GrantLoginReward(ctx, userID, cfg.LoginReward, today)
	return err
}

// acquireDrawToken Redis SetNX 实现 1 次/秒限流；Redis 不可用时放行。
func (s *LotteryService) acquireDrawToken(ctx context.Context, userID int64) bool {
	if s.drawLimiter == nil {
		return true
	}
	return s.drawLimiter.TryAcquireDrawToken(ctx, userID)
}

// invalidateBalanceCache 失效用户余额缓存（best-effort）。
func (s *LotteryService) invalidateBalanceCache(ctx context.Context, userID int64) {
	if s.billingCacheService == nil {
		return
	}
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.billingCacheService.InvalidateUserBalance(cacheCtx, userID)
	}()
}

// uniformPrize 在 [min, max] 内均匀分布随机金额，保留 2 位小数。
func uniformPrize(min, max float64) float64 {
	if max <= min {
		return math.Round(min*100) / 100
	}
	v := min + rand.Float64()*(max-min)
	return math.Round(v*100) / 100
}
