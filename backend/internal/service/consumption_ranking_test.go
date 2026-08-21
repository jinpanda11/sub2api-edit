package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newRankingTestEnv(t *testing.T) *ConsumptionRankingService {
	t.Helper()

	// SQLite is used here to exercise the persisted usage_logs aggregation path.
	db, err := sql.Open("sqlite", "file:consumption_ranking_test?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(entsql.OpenDB(dialect.SQLite, db))))
	t.Cleanup(func() { _ = client.Close() })

	return NewConsumptionRankingService(client, nil, nil)
}
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

func TestGetToday_FromUsageLogs(t *testing.T) {
	svc := newRankingTestEnv(t)
	ctx := context.Background()

	userA := mustInsertRankingUser(t, ctx, svc.entClient, "ab123456cd@qq.com")
	userB := mustInsertRankingUser(t, ctx, svc.entClient, "testuser@example.com")
	accountA := mustInsertRankingAccount(t, ctx, svc.entClient, "ranking-account-a")
	accountB := mustInsertRankingAccount(t, ctx, svc.entClient, "ranking-account-b")
	keyA := mustInsertRankingAPIKey(t, ctx, svc.entClient, userA, accountA, "ranking-key-a")
	keyB := mustInsertRankingAPIKey(t, ctx, svc.entClient, userB, accountB, "ranking-key-b")
	today := time.Now().In(shanghaiLoc).Truncate(time.Hour)

	mustInsertRankingUsageLog(t, ctx, svc.entClient, userA, keyA, accountA, "ranking-a-1", today, BillingTypeBalance, 2.25, 100, 20, 30, 40)
	mustInsertRankingUsageLog(t, ctx, svc.entClient, userA, keyA, accountA, "ranking-a-2", today.Add(time.Minute), BillingTypeBalance, 1.75, 10, 5, 5, 5)
	mustInsertRankingUsageLog(t, ctx, svc.entClient, userA, keyA, accountA, "ranking-a-free", today.Add(2*time.Minute), BillingTypeBalance, 0, 999, 999, 999, 999)
	mustInsertRankingUsageLog(t, ctx, svc.entClient, userB, keyB, accountB, "ranking-b-subscription", today.Add(3*time.Minute), BillingTypeSubscription, 99, 500, 0, 0, 0)
	mustInsertRankingUsageLog(t, ctx, svc.entClient, userB, keyB, accountB, "ranking-b-1", today.Add(4*time.Minute), BillingTypeBalance, 3.00, 20, 10, 5, 5)

	resp, err := svc.GetToday(ctx)
	require.NoError(t, err)
	require.Equal(t, shanghaiDateStr(time.Now()), resp.Date)
	require.Len(t, resp.List, 2)
	require.Equal(t, 1, resp.List[0].Rank)
	require.Equal(t, "ab******cd@qq.com", resp.List[0].MaskedEmail)
	require.Equal(t, 4.0, resp.List[0].Amount)
	require.Equal(t, int64(215), resp.List[0].TotalTokens)
	require.Equal(t, 2, resp.List[1].Rank)
	require.Equal(t, "te****er@example.com", resp.List[1].MaskedEmail)
	require.Equal(t, 3.0, resp.List[1].Amount)
	require.Equal(t, int64(40), resp.List[1].TotalTokens)

	mustInsertRankingUsageLog(t, ctx, svc.entClient, userA, keyA, accountA, "ranking-a-3", today.Add(5*time.Minute), BillingTypeBalance, 10.0, 1, 2, 3, 4)
	resp, err = svc.GetToday(ctx)
	require.NoError(t, err)
	require.Equal(t, 14.0, resp.List[0].Amount)
	require.Equal(t, int64(225), resp.List[0].TotalTokens)
}

func mustInsertRankingAccount(t *testing.T, ctx context.Context, client *dbent.Client, name string) int64 {
	t.Helper()
	a, err := client.Account.Create().
		SetName(name).
		SetPlatform(PlatformOpenAI).
		SetType(AccountTypeAPIKey).
		SetStatus(StatusActive).
		SetCredentials(map[string]any{"api_key": "test"}).
		Save(ctx)
	require.NoError(t, err)
	return a.ID
}

func mustInsertRankingAPIKey(t *testing.T, ctx context.Context, client *dbent.Client, userID, accountID int64, key string) int64 {
	t.Helper()
	k, err := client.APIKey.Create().
		SetUserID(userID).
		SetKey(key).
		SetName(key).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)
	return k.ID
}

func mustInsertRankingUsageLog(t *testing.T, ctx context.Context, client *dbent.Client, userID, keyID, accountID int64, requestID string, createdAt time.Time, billingType int8, actualCost float64, input, output, cacheCreation, cacheRead int) {
	t.Helper()
	_, err := client.UsageLog.Create().
		SetUserID(userID).
		SetAPIKeyID(keyID).
		SetAccountID(accountID).
		SetRequestID(requestID).
		SetModel("gpt-5").
		SetBillingType(billingType).
		SetActualCost(actualCost).
		SetTotalCost(actualCost).
		SetInputTokens(input).
		SetOutputTokens(output).
		SetCacheCreationTokens(cacheCreation).
		SetCacheReadTokens(cacheRead).
		SetCreatedAt(createdAt).
		Save(ctx)
	require.NoError(t, err)
}
