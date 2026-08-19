package service

import (
	"context"
	"database/sql"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/redis/go-redis/v9"
	_ "modernc.org/sqlite"
)

func TestMaskRankingEmail(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// 长度自适应（伪代码：local[:2] + (len-4)个星号 + local[-2:]）
		{"ab123456cd@qq.com", "ab******cd@qq.com"},
		{"testuser@example.com", "te****er@example.com"},
		{"a@b.com", "a***@b.com"},
		{"abcd@x.com", "a***@x.com"},
		{"", "***@***"},
		{"no-at-sign", "***@***"},
		{"user@", "***@***"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, MaskRankingEmail(c.in), "input=%q", c.in)
	}
}

func TestRound4(t *testing.T) {
	require.Equal(t, 128.4567, round4(128.45673))
	require.Equal(t, 95.32, round4(95.32))
	require.Equal(t, 0.0, round4(0.00001))
}

func newRankingTestEnv(t *testing.T) (*ConsumptionRankingService, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	// sqlite ent client 仅用于 fetchEmails（用户邮箱批量查询）。
	db, err := sql.Open("sqlite", "file:consumption_ranking_test?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(entsql.OpenDB(dialect.SQLite, db))))
	t.Cleanup(func() { _ = client.Close() })

	return NewConsumptionRankingService(client, rdb, nil), mr
}

func mustInsertRankingUser(t *testing.T, ctx context.Context, client *dbent.Client, email string) int64 {
	t.Helper()
	u, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)
	return u.ID
}

func TestGetToday_FromWarmRedis(t *testing.T) {
	svc, mr := newRankingTestEnv(t)
	ctx := context.Background()

	userA := mustInsertRankingUser(t, ctx, svc.entClient, "ab123456cd@qq.com")
	userB := mustInsertRankingUser(t, ctx, svc.entClient, "testuser@example.com")
	date := shanghaiDateStr(time.Now())

	// 预置 Redis：warm 标记 + 两个用户的分数
	key := rankingRedisKey(date)
	mr.ZAdd(key, 128.4567, int64ToStr(userA))
	mr.ZAdd(key, 95.32, int64ToStr(userB))
	mr.Set(rankingWarmKey(date), "1")

	resp, err := svc.GetToday(ctx)
	require.NoError(t, err)
	require.Equal(t, date, resp.Date)
	require.Len(t, resp.List, 2)
	require.Equal(t, 1, resp.List[0].Rank)
	require.Equal(t, "ab******cd@qq.com", resp.List[0].MaskedEmail)
	require.Equal(t, 128.4567, resp.List[0].Amount)
	require.Equal(t, 2, resp.List[1].Rank)
	require.Equal(t, "te****er@example.com", resp.List[1].MaskedEmail)
	require.Equal(t, 95.32, resp.List[1].Amount)
}

func TestIncrementAsync_ZIncrBy(t *testing.T) {
	svc, mr := newRankingTestEnv(t)
	userID := int64(42)
	key := rankingRedisKey(shanghaiDateStr(time.Now()))

	svc.IncrementAsync(userID, 1.25)
	svc.IncrementAsync(userID, 0.75)
	require.Eventually(t, func() bool {
		score, _ := mr.ZScore(key, int64ToStr(userID))
		return score == 2.0
	}, 2*time.Second, 10*time.Millisecond)
}

func int64ToStr(v int64) string {
	return strconv.FormatInt(v, 10)
}
