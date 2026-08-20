-- 每日盲盒抽奖：标记可用次数所归属的上海自然日。
-- 用于跨天时清空未使用次数（未使用次数不跨天保留）。
ALTER TABLE lottery_chances ADD COLUMN IF NOT EXISTS count_date VARCHAR(10);

CREATE INDEX IF NOT EXISTS idx_lottery_chances_count_date ON lottery_chances (count_date);

COMMENT ON COLUMN lottery_chances.count_date IS 'available_count 所归属的上海自然日（YYYY-MM-DD），用于跨天时清空未使用次数。';
