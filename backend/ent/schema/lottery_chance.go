package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

// LotteryChance 用户盲盒抽奖次数账户。
//
// 删除策略：硬删除
// 抽奖次数是纯增量业务数据，用户注销后无需保留；且本表不承载审计所需的
// 历史语义（抽奖明细在 lottery_records）。
type LotteryChance struct {
	ent.Schema
}

func (LotteryChance) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "lottery_chances"},
	}
}

func (LotteryChance) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (LotteryChance) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Unique(),
		// 当前可用次数：跨天保留，无上限，仅抽奖消耗。
		field.Int("available_count").
			Default(0),
		// 今日（Asia/Shanghai）已通过充值/兑换码来源获得的次数（0 ~ recharge_daily_max）。
		field.Int("today_recharge_count").
			Default(0),
		// recharge_count 所属的上海自然日（YYYY-MM-DD），用于每日 00:00 惰性重置桶。
		field.String("recharge_date").
			Optional().
			Nillable().
			MaxLen(10),
		// 最近一次发放登录奖励的上海自然日（YYYY-MM-DD），保证每日只发一次。
		field.String("last_login_date").
			Optional().
			Nillable().
			MaxLen(10),
	}
}

func (LotteryChance) Edges() []ent.Edge {
	return nil
}

func (LotteryChance) Indexes() []ent.Index {
	return []ent.Index{
		// user_id 已在 Fields() 中声明 Unique()，无需重复索引
		index.Fields("recharge_date"),
		index.Fields("last_login_date"),
	}
}
