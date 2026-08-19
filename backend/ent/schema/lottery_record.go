package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

// LotteryRecord 盲盒抽奖明细记录。
//
// 删除策略：硬删除
// 抽奖记录仅用于用户历史展示与运营统计，与用户生命周期无关，
// 注销用户时一并清理即可，无需保留软删除语义。
type LotteryRecord struct {
	ent.Schema
}

func (LotteryRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "lottery_records"},
	}
}

func (LotteryRecord) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (LotteryRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		// 中奖金额（与余额同单位，页面展示为 US$）。
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		// 抽奖后账户余额，便于审计与对账。
		field.Float("balance_after").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		// 消耗次数的来源：login / recharge，用于运营统计。
		field.String("source").
			MaxLen(20).
			Default("login"),
	}
}

func (LotteryRecord) Edges() []ent.Edge {
	return nil
}

func (LotteryRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("user_id", "created_at"),
		index.Fields("created_at"),
	}
}
