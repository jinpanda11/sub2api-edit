package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/lotterychance"
	"github.com/Wei-Shaw/sub2api/ent/lotteryrecord"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// lotteryChanceRepository implements service.LotteryChanceRepository.
//
// 所有写操作都通过 clientFromContext 复用调用方事务（ent Tx），保证与
// 余额变更、兑换码标记等操作在同一事务内原子提交。
type lotteryChanceRepository struct {
	client *ent.Client
}

func NewLotteryChanceRepository(client *ent.Client) service.LotteryChanceRepository {
	return &lotteryChanceRepository{client: client}
}

func (r *lotteryChanceRepository) GetOrCreate(ctx context.Context, userID int64) (*service.LotteryChance, error) {
	client := clientFromContext(ctx, r.client)
	err := client.LotteryChance.Create().
		SetUserID(userID).
		SetAvailableCount(0).
		SetTodayRechargeCount(0).
		OnConflictColumns(lotterychance.FieldUserID).
		DoNothing().
		Exec(ctx)
	// ent 在冲突（行已存在）时对 DO NOTHING 的 INSERT 返回 sql.ErrNoRows，
	// 按项目惯例视为成功，随后走查询路径返回现有行。
	if err != nil && !isSQLNoRowsError(err) {
		return nil, err
	}
	return r.Get(ctx, userID)
}

func (r *lotteryChanceRepository) Get(ctx context.Context, userID int64) (*service.LotteryChance, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.LotteryChance.Query().
		Where(lotterychance.UserIDEQ(userID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, service.ErrLotteryChanceNotFound
		}
		return nil, err
	}
	return lotteryChanceEntityToService(m), nil
}

// ResetDailyCountIfExpired 若可用次数所属上海自然日（count_date）早于 today 或尚未标记，
// 则清零可用次数并标记为 today。
// 实现“未使用次数不跨天保留”：跨天后旧次数一律作废。
func (r *lotteryChanceRepository) ResetDailyCountIfExpired(ctx context.Context, userID int64, today string) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.LotteryChance.Update().
		Where(
			lotterychance.UserIDEQ(userID),
			lotterychance.Or(
				lotterychance.CountDateIsNil(),
				lotterychance.CountDateNEQ(today),
			),
		).
		SetAvailableCount(0).
		SetCountDate(today).
		Save(ctx)
	return err
}

// GrantLoginReward 原子发放登录奖励，仅当 last_login_date < today 时生效（返回是否发放）。
// 并发登录时依赖 WHERE 条件保证每日最多发放一次。
func (r *lotteryChanceRepository) GrantLoginReward(ctx context.Context, userID int64, reward int, today string) (bool, error) {
	client := clientFromContext(ctx, r.client)
	n, err := client.LotteryChance.Update().
		Where(
			lotterychance.UserIDEQ(userID),
			lotterychance.Or(
				lotterychance.LastLoginDateIsNil(),
				lotterychance.LastLoginDateLT(today),
			),
		).
		AddAvailableCount(reward).
		SetLastLoginDate(today).
		Save(ctx)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GrantRechargeReward 原子发放充值/兑换码奖励。
// 若 recharge_date 非今日（Asia/Shanghai）则先重置当日桶；实际发放数受每日上限约束。
func (r *lotteryChanceRepository) GrantRechargeReward(ctx context.Context, userID int64, earned int, today string, dailyMax int) (int, error) {
	client := clientFromContext(ctx, r.client)

	// 行锁串行化同一用户并发入账，防止超出每日上限。
	row, err := client.LotteryChance.Query().
		Where(lotterychance.UserIDEQ(userID)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return 0, err
	}

	current := row.TodayRechargeCount
	if row.RechargeDate == nil || *row.RechargeDate != today {
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

	_, err = client.LotteryChance.Update().
		Where(lotterychance.UserIDEQ(userID)).
		AddAvailableCount(add).
		SetTodayRechargeCount(current + add).
		SetRechargeDate(today).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	return add, nil
}

// ConsumeChance 原子消耗 1 次可用次数，无可用次数时返回 false。
func (r *lotteryChanceRepository) ConsumeChance(ctx context.Context, userID int64) (bool, error) {
	client := clientFromContext(ctx, r.client)
	n, err := client.LotteryChance.Update().
		Where(
			lotterychance.UserIDEQ(userID),
			lotterychance.AvailableCountGT(0),
		).
		AddAvailableCount(-1).
		Save(ctx)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// lotteryRecordRepository implements service.LotteryRecordRepository.
type lotteryRecordRepository struct {
	client *ent.Client
}

func NewLotteryRecordRepository(client *ent.Client) service.LotteryRecordRepository {
	return &lotteryRecordRepository{client: client}
}

func (r *lotteryRecordRepository) Create(ctx context.Context, record *service.LotteryRecord) (int64, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.LotteryRecord.Create().
		SetUserID(record.UserID).
		SetAmount(record.Amount).
		SetBalanceAfter(record.BalanceAfter).
		SetSource(record.Source).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	return m.ID, nil
}

func (r *lotteryRecordRepository) UpdateBalanceAfter(ctx context.Context, id int64, balanceAfter float64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.LotteryRecord.UpdateOneID(id).
		SetBalanceAfter(balanceAfter).
		Save(ctx)
	return err
}

func (r *lotteryRecordRepository) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]service.LotteryRecord, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.LotteryRecord.Query().
		Where(lotteryrecord.UserIDEQ(userID)).
		Order(ent.Desc(lotteryrecord.FieldCreatedAt)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.LotteryRecord, 0, len(rows))
	for i := range rows {
		out = append(out, *lotteryRecordEntityToService(rows[i]))
	}
	return out, nil
}

func (r *lotteryRecordRepository) CountByUser(ctx context.Context, userID int64) (int64, error) {
	client := clientFromContext(ctx, r.client)
	n, err := client.LotteryRecord.Query().
		Where(lotteryrecord.UserIDEQ(userID)).
		Count(ctx)
	return int64(n), err
}

// DailyAggregate 按上海自然日聚合 [from, to) 区间内的抽奖记录，并返回区间内去重参与人数。
// 使用 ent client 暴露的原始 QueryContext 执行原生聚合 SQL，避免新增数据库连接依赖。
func (r *lotteryRecordRepository) DailyAggregate(ctx context.Context, from, to time.Time) ([]service.LotteryDailyAggregate, int64, error) {
	const query = `
SELECT
	TO_CHAR(created_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD') AS date,
	COUNT(*) AS draws,
	COUNT(DISTINCT user_id) AS participants,
	COALESCE(SUM(amount), 0) AS total_amount
FROM lottery_records
WHERE created_at >= $1 AND created_at < $2
GROUP BY date
ORDER BY date ASC`

	rows, err := r.client.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, 0, fmt.Errorf("query lottery daily aggregate: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []service.LotteryDailyAggregate
	for rows.Next() {
		var (
			date         string
			draws        int64
			participants int64
			totalAmount  sql.NullFloat64
		)
		if err := rows.Scan(&date, &draws, &participants, &totalAmount); err != nil {
			return nil, 0, fmt.Errorf("scan lottery daily aggregate: %w", err)
		}
		amt := 0.0
		if totalAmount.Valid {
			amt = totalAmount.Float64
		}
		out = append(out, service.LotteryDailyAggregate{
			Date:         date,
			Draws:        draws,
			Participants: participants,
			TotalAmount:  amt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate lottery daily aggregate: %w", err)
	}

	// 区间内去重参与人数（与每日去重口径一致，避免跨天重复计数）。
	rows2, err := r.client.QueryContext(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM lottery_records WHERE created_at >= $1 AND created_at < $2`,
		from, to,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("count lottery distinct participants: %w", err)
	}
	defer func() { _ = rows2.Close() }()
	var rangeParticipants int64
	if rows2.Next() {
		if err := rows2.Scan(&rangeParticipants); err != nil {
			return nil, 0, fmt.Errorf("scan lottery distinct participants: %w", err)
		}
	}
	if err := rows2.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate lottery distinct participants: %w", err)
	}

	return out, rangeParticipants, nil
}

func lotteryChanceEntityToService(m *ent.LotteryChance) *service.LotteryChance {
	return &service.LotteryChance{
		ID:                 m.ID,
		UserID:             m.UserID,
		AvailableCount:     m.AvailableCount,
		TodayRechargeCount: m.TodayRechargeCount,
		RechargeDate:       m.RechargeDate,
		LastLoginDate:      m.LastLoginDate,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

func lotteryRecordEntityToService(m *ent.LotteryRecord) *service.LotteryRecord {
	return &service.LotteryRecord{
		ID:           m.ID,
		UserID:       m.UserID,
		Amount:       m.Amount,
		BalanceAfter: m.BalanceAfter,
		Source:       m.Source,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
