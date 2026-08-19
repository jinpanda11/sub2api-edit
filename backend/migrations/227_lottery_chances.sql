-- 每日盲盒抽奖：用户抽奖次数账户（跨天保留，无总上限）。

CREATE TABLE IF NOT EXISTS lottery_chances (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    available_count INT NOT NULL DEFAULT 0,
    today_recharge_count INT NOT NULL DEFAULT 0,
    recharge_date VARCHAR(10),
    last_login_date VARCHAR(10),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE lottery_chances IS '用户盲盒抽奖次数账户。';
COMMENT ON COLUMN lottery_chances.recharge_date IS 'today_recharge_count 所属的上海自然日（YYYY-MM-DD），用于每日 00:00 惰性重置。';
COMMENT ON COLUMN lottery_chances.last_login_date IS '最近一次发放登录奖励的上海自然日（YYYY-MM-DD），保证每日只发一次。';

CREATE INDEX IF NOT EXISTS idx_lottery_chances_recharge_date ON lottery_chances (recharge_date);
CREATE INDEX IF NOT EXISTS idx_lottery_chances_last_login_date ON lottery_chances (last_login_date);
