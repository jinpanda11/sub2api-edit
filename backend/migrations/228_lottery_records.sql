-- 每日盲盒抽奖：中奖明细记录。

CREATE TABLE IF NOT EXISTS lottery_records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    amount DECIMAL(20, 8) NOT NULL DEFAULT 0,
    balance_after DECIMAL(20, 8) NOT NULL DEFAULT 0,
    source VARCHAR(20) NOT NULL DEFAULT 'login',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE lottery_records IS '盲盒抽奖中奖明细。';
COMMENT ON COLUMN lottery_records.balance_after IS '抽奖后账户余额，便于审计与对账。';
COMMENT ON COLUMN lottery_records.source IS '记录来源，当前恒为 draw（次数来源不做溯源）。';

CREATE INDEX IF NOT EXISTS idx_lottery_records_user_id ON lottery_records (user_id);
CREATE INDEX IF NOT EXISTS idx_lottery_records_user_created ON lottery_records (user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_lottery_records_created_at ON lottery_records (created_at);
