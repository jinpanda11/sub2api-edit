package service

import (
	"context"
	"database/sql"
	"math"
	"sort"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

// ---------- fake repositories ----------

type fakeChanceRepo struct {
	row            *LotteryChance
	countDate      *string
	grantsLogin    int
	grantsRecharge int
	consumed       int
	rechargeEarned []int
}

func (f *fakeChanceRepo) GetOrCreate(ctx context.Context, userID int64) (*LotteryChance, error) {
	if f.row == nil {
		f.row = &LotteryChance{UserID: userID}
	}
	return f.row, nil
}

func (f *fakeChanceRepo) ResetDailyCountIfExpired(ctx context.Context, userID int64, today string) error {
	if f.row == nil {
		return nil
	}
	if f.countDate != nil && *f.countDate == today {
		return nil
	}
	f.row.AvailableCount = 0
	f.countDate = &today
	return nil
}

func (f *fakeChanceRepo) Get(ctx context.Context, userID int64) (*LotteryChance, error) {
	if f.row == nil {
		return nil, ErrLotteryChanceNotFound
	}
	return f.row, nil
}

func (f *fakeChanceRepo) GrantLoginReward(ctx context.Context, userID int64, reward int, today string) (bool, error) {
	f.grantsLogin++
	if f.row.LastLoginDate != nil && *f.row.LastLoginDate == today {
		return false, nil
	}
	f.row.AvailableCount += reward
	f.row.LastLoginDate = &today
	return true, nil
}

func (f *fakeChanceRepo) GrantRechargeReward(ctx context.Context, userID int64, earned int, today string, dailyMax int) (int, error) {
	f.grantsRecharge++
	f.rechargeEarned = append(f.rechargeEarned, earned)
	current := f.row.TodayRechargeCount
	if f.row.RechargeDate == nil || *f.row.RechargeDate != today {
		current = 0
	}
	remaining := dailyMax - current
	if remaining <= 0 {
		return 0, nil
	}
	add := earned
	if add > remaining {
		add = remaining
	}
	f.row.AvailableCount += add
	f.row.TodayRechargeCount = current + add
	f.row.RechargeDate = &today
	return add, nil
}

func (f *fakeChanceRepo) ConsumeChance(ctx context.Context, userID int64) (bool, error) {
	if f.row == nil || f.row.AvailableCount <= 0 {
		return false, nil
	}
	f.row.AvailableCount--
	f.consumed++
	return true, nil
}

// fakeUserRepo 嵌入 UserRepository 接口满足全部方法，仅覆盖测试用到的两个。
type fakeUserRepo struct {
	UserRepository
	user *User
}

func (f *fakeUserRepo) GetByID(ctx context.Context, id int64) (*User, error) {
	if f.user == nil {
		return nil, ErrUserNotFound
	}
	return f.user, nil
}

func (f *fakeUserRepo) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	if f.user == nil {
		return ErrUserNotFound
	}
	f.user.Balance += amount
	return nil
}

type fakeRecordRepo struct {
	records []LotteryRecord
}

func (f *fakeRecordRepo) Create(ctx context.Context, record *LotteryRecord) (int64, error) {
	record.ID = int64(len(f.records) + 1)
	f.records = append(f.records, *record)
	return record.ID, nil
}

func (f *fakeRecordRepo) UpdateBalanceAfter(ctx context.Context, id int64, balanceAfter float64) error {
	for i := range f.records {
		if f.records[i].ID == id {
			f.records[i].BalanceAfter = balanceAfter
			return nil
		}
	}
	return nil
}

func (f *fakeRecordRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]LotteryRecord, error) {
	out := make([]LotteryRecord, 0, len(f.records))
	for _, r := range f.records {
		if r.UserID == userID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRecordRepo) CountByUser(ctx context.Context, userID int64) (int64, error) {
	return int64(len(f.records)), nil
}

func (f *fakeRecordRepo) DailyAggregate(ctx context.Context, from, to time.Time) ([]LotteryDailyAggregate, int64, error) {
	byDate := map[string]LotteryDailyAggregate{}
	participants := map[string]map[int64]struct{}{}
	rangeParticipants := map[int64]struct{}{}
	for _, r := range f.records {
		key := shanghaiStartOfDay(r.CreatedAt)
		if key.Before(from) || !key.Before(to) {
			continue
		}
		date := key.Format("2006-01-02")
		agg := byDate[date]
		agg.Date = date
		agg.Draws++
		agg.TotalAmount += r.Amount
		byDate[date] = agg
		if participants[date] == nil {
			participants[date] = map[int64]struct{}{}
		}
		participants[date][r.UserID] = struct{}{}
		rangeParticipants[r.UserID] = struct{}{}
	}
	out := make([]LotteryDailyAggregate, 0, len(byDate))
	for date, agg := range byDate {
		agg.Participants = int64(len(participants[date]))
		out = append(out, agg)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out, int64(len(rangeParticipants)), nil
}

func newTestLotteryService(entClient *dbent.Client, chanceRepo LotteryChanceRepository, userRepo UserRepository, recordRepo LotteryRecordRepository) *LotteryService {
	if userRepo == nil {
		userRepo = &fakeUserRepo{}
	}
	if chanceRepo == nil {
		chanceRepo = &fakeChanceRepo{}
	}
	if recordRepo == nil {
		recordRepo = &fakeRecordRepo{}
	}
	return NewLotteryService(entClient, userRepo, chanceRepo, recordRepo, nil, nil, nil)
}

func newSQLiteEntClient(t *testing.T) *dbent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", "file:lottery_service_test?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// ---------- tests ----------

func TestUniformPrizeStaysInRangeAndRoundsToCents(t *testing.T) {
	for i := 0; i < 500; i++ {
		v := uniformPrize(0.10, 2.00)
		require.GreaterOrEqual(t, v, 0.10)
		require.LessOrEqual(t, v, 2.00)
		require.InDelta(t, v, math.Round(v*100)/100, 1e-9)
	}
	// min == max 边界
	require.Equal(t, 0.5, uniformPrize(0.5, 0.5))
}

func TestGrantLoginReward_OncePerShanghaiDay(t *testing.T) {
	svc := newTestLotteryService(nil, nil, nil, nil)
	chanceRepo, ok := svc.chanceRepo.(*fakeChanceRepo)
	require.True(t, ok)

	require.NoError(t, svc.GrantLoginReward(context.Background(), 1, 1))
	require.Equal(t, 1, chanceRepo.row.AvailableCount)
	require.Equal(t, shanghaiDateStr(time.Now()), *chanceRepo.row.LastLoginDate)

	// 同日再次发放应被去重
	require.NoError(t, svc.GrantLoginReward(context.Background(), 1, 1))
	require.Equal(t, 1, chanceRepo.row.AvailableCount)
}

func TestGrantRechargeReward_FloorAndDailyCap(t *testing.T) {
	svc := newTestLotteryService(nil, nil, nil, nil)
	chanceRepo, ok := svc.chanceRepo.(*fakeChanceRepo)
	require.True(t, ok)

	// 充值 25 → floor(25/10)=2
	require.NoError(t, svc.GrantRechargeReward(context.Background(), 1, 25))
	require.Equal(t, 2, chanceRepo.row.AvailableCount)
	require.Equal(t, 2, chanceRepo.row.TodayRechargeCount)

	// 再充值 40 → 理论 +4，但当日上限 5 只补 3
	require.NoError(t, svc.GrantRechargeReward(context.Background(), 1, 40))
	require.Equal(t, 5, chanceRepo.row.TodayRechargeCount)
	require.Equal(t, 5, chanceRepo.row.AvailableCount)

	// 超出上限后不再发放
	require.NoError(t, svc.GrantRechargeReward(context.Background(), 1, 100))
	require.Equal(t, 5, chanceRepo.row.TodayRechargeCount)
	require.Equal(t, 5, chanceRepo.row.AvailableCount)

	// 小额充值不足一个单位 → 0
	require.NoError(t, svc.GrantRechargeReward(context.Background(), 1, 9.99))
	require.Equal(t, 5, chanceRepo.row.AvailableCount)
}

func TestGrantRechargeReward_DailyBucketResetsOnShanghaiDateChange(t *testing.T) {
	svc := newTestLotteryService(nil, nil, nil, nil)
	chanceRepo, ok := svc.chanceRepo.(*fakeChanceRepo)
	require.True(t, ok)

	require.NoError(t, svc.GrantRechargeReward(context.Background(), 1, 60))
	require.Equal(t, 5, chanceRepo.row.AvailableCount)

	// 模拟进入新的一天：直接改桶日期，下次发放应重置桶并重新计满 5
	yesterday := "2000-01-01"
	chanceRepo.row.RechargeDate = &yesterday
	chanceRepo.row.TodayRechargeCount = 5

	require.NoError(t, svc.GrantRechargeReward(context.Background(), 1, 40))
	require.Equal(t, 4, chanceRepo.row.TodayRechargeCount)
	require.Equal(t, 5+4, chanceRepo.row.AvailableCount)
	require.Equal(t, shanghaiDateStr(time.Now()), *chanceRepo.row.RechargeDate)
}

func TestDraw_NoChancesReturnsError(t *testing.T) {
	client := newSQLiteEntClient(t)
	svc := newTestLotteryService(client, nil, nil, nil)

	_, err := svc.Draw(context.Background(), 1)
	require.ErrorIs(t, err, ErrLotteryNoChances)
}

func TestDraw_SuccessConsumesChanceAndCreditsBalance(t *testing.T) {
	client := newSQLiteEntClient(t)
	today := shanghaiDateStr(time.Now())
	chanceRepo := &fakeChanceRepo{
		row: &LotteryChance{UserID: 1, AvailableCount: 3, TodayRechargeCount: 0},
		// 标记为今日次数，重置逻辑不会误清空今日有效次数。
		countDate: &today,
	}
	userRepo := &fakeUserRepo{user: &User{ID: 1, Balance: 10.0}}
	recordRepo := &fakeRecordRepo{}
	svc := newTestLotteryService(client, chanceRepo, userRepo, recordRepo)

	result, err := svc.Draw(context.Background(), 1)
	require.NoError(t, err)
	require.GreaterOrEqual(t, result.PrizeAmount, 0.10)
	require.LessOrEqual(t, result.PrizeAmount, 2.00)
	require.InDelta(t, result.NewBalance, 10.0+result.PrizeAmount, 1e-9)
	require.Equal(t, 2, result.RemainingCount)
	require.Equal(t, 1, chanceRepo.consumed)
	require.Len(t, recordRepo.records, 1)
	require.Equal(t, result.PrizeAmount, recordRepo.records[0].Amount)
	require.Equal(t, "draw", recordRepo.records[0].Source)
	require.InDelta(t, result.NewBalance, recordRepo.records[0].BalanceAfter, 1e-9)
}

func TestDailyCountReset_ExpiresUnusedChancesAcrossDays(t *testing.T) {
	userRepo := &fakeUserRepo{user: &User{ID: 1}}
	svc := newTestLotteryService(nil, nil, userRepo, nil)
	chanceRepo, ok := svc.chanceRepo.(*fakeChanceRepo)
	require.True(t, ok)

	today := shanghaiDateStr(time.Now())
	yesterday := time.Now().In(shanghaiLoc).AddDate(0, 0, -1).Format("2006-01-02")

	// 模拟昨日登录获得 +1（未抽奖），对应行尚无 count_date 标记。
	_, err := chanceRepo.GetOrCreate(context.Background(), 1)
	require.NoError(t, err)
	_, err = chanceRepo.GrantLoginReward(context.Background(), 1, 1, yesterday)
	require.NoError(t, err)
	require.Equal(t, 1, chanceRepo.row.AvailableCount)

	// 进入新的一天调用 Status，应清空昨日未使用次数并标记今日。
	status, err := svc.Status(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, 0, status.AvailableCount)
	require.Equal(t, today, *chanceRepo.countDate)
}

func TestDraw_DisabledReturnsError(t *testing.T) {
	svc := newTestLotteryService(nil, nil, nil, nil)
	settingRepo := &lotteryTestSettingRepo{}
	settingRepo.values = map[string]string{SettingKeyLotteryEnabled: "false"}
	svc.settingRepo = settingRepo

	_, err := svc.Draw(context.Background(), 1)
	require.ErrorIs(t, err, ErrLotteryDisabled)
}

// lotteryTestSettingRepo 供配置读取测试使用。
type lotteryTestSettingRepo struct {
	values map[string]string
}

func (f *lotteryTestSettingRepo) Get(ctx context.Context, key string) (*Setting, error) {
	panic("not implemented")
}
func (f *lotteryTestSettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	panic("not implemented")
}
func (f *lotteryTestSettingRepo) Set(ctx context.Context, key, value string) error {
	panic("not implemented")
}
func (f *lotteryTestSettingRepo) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := f.values[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}
func (f *lotteryTestSettingRepo) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("not implemented")
}
func (f *lotteryTestSettingRepo) GetAll(ctx context.Context) (map[string]string, error) {
	return f.values, nil
}
func (f *lotteryTestSettingRepo) Delete(ctx context.Context, key string) error {
	panic("not implemented")
}

var _ SettingRepository = (*lotteryTestSettingRepo)(nil)

func TestAdminDailyStats_AggregatesAndFillsZeros(t *testing.T) {
	from, _ := time.ParseInLocation("2006-01-02", "2026-08-17", shanghaiLoc)
	records := []LotteryRecord{
		{UserID: 1, Amount: 1.00, CreatedAt: from.Add(10 * time.Hour)},                 // 08-17 user1
		{UserID: 1, Amount: 2.00, CreatedAt: from.Add(11 * time.Hour)},                 // 08-17 user1（同日多次）
		{UserID: 2, Amount: 0.50, CreatedAt: from.AddDate(0, 0, 1).Add(5 * time.Hour)}, // 08-18 user2
		{UserID: 2, Amount: 1.50, CreatedAt: from.AddDate(0, 0, 3).Add(5 * time.Hour)}, // 08-20 user2
	}
	recordRepo := &fakeRecordRepo{records: records}
	svc := newTestLotteryService(nil, nil, nil, recordRepo)

	stats, err := svc.AdminDailyStats(context.Background(), "2026-08-17", "2026-08-20")
	require.NoError(t, err)
	require.Equal(t, "2026-08-17", stats.StartDate)
	require.Equal(t, "2026-08-20", stats.EndDate)

	require.Len(t, stats.Daily, 4)
	require.Equal(t, "2026-08-17", stats.Daily[0].Date)
	require.Equal(t, int64(2), stats.Daily[0].Draws)
	require.Equal(t, int64(1), stats.Daily[0].Participants)
	require.InDelta(t, 3.0, stats.Daily[0].TotalAmount, 1e-9)

	// 无数据日期补齐为零。
	require.Equal(t, "2026-08-19", stats.Daily[2].Date)
	require.Equal(t, int64(0), stats.Daily[2].Draws)
	require.Equal(t, int64(0), stats.Daily[2].Participants)
	require.InDelta(t, 0.0, stats.Daily[2].TotalAmount, 1e-9)

	require.Equal(t, int64(4), stats.Summary.TotalDraws)
	require.Equal(t, int64(2), stats.Summary.TotalParticipants)
	require.InDelta(t, 5.0, stats.Summary.TotalAmount, 1e-9)
	require.InDelta(t, 1.25, stats.Summary.AvgAmount, 1e-9)
}

func TestAdminDailyStats_InvalidDateReturnsError(t *testing.T) {
	recordRepo := &fakeRecordRepo{}
	svc := newTestLotteryService(nil, nil, nil, recordRepo)
	_, err := svc.AdminDailyStats(context.Background(), "2026-13-99", "2026-08-20")
	require.Error(t, err)
}

func TestResolveLotteryStatsWindow_DefaultsToLast30Days(t *testing.T) {
	start, end, err := resolveLotteryStatsWindow("", "")
	require.NoError(t, err)
	require.True(t, start.Before(end))
	// 默认窗口为 30 天（含当天），end/start 跨 30 个自然日。
	require.Equal(t, 30, int(end.Sub(start)/(24*time.Hour)))
	// start 与 end 都对齐到上海自然日的 00:00。
	require.Equal(t, 0, start.Hour())
	require.Equal(t, 0, start.Minute())
	require.Equal(t, 0, end.Hour())
}

func TestResolveLotteryStatsWindow_ExplicitRange(t *testing.T) {
	start, end, err := resolveLotteryStatsWindow("2026-08-01", "2026-08-03")
	require.NoError(t, err)
	require.Equal(t, "2026-08-01", start.Format("2006-01-02"))
	// end 为闭区间 08-03，聚合上界为次日 08-04。
	require.Equal(t, "2026-08-04", end.Format("2006-01-02"))
	require.Equal(t, 3, int(end.Sub(start)/(24*time.Hour)))
}
