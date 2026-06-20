-- 修复 wallet_trades 重复/冗余列

-- 若 GORM AutoMigrate 曾创建 pnlusd 列，迁移数据后删除
SET @has_pnlusd := (
    SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'wallet_trades'
      AND COLUMN_NAME = 'pnlusd'
);
SET @migrate_pnl := IF(
    @has_pnlusd > 0,
    'UPDATE wallet_trades SET pnl_usd = COALESCE(pnl_usd, pnlusd) WHERE pnlusd IS NOT NULL',
    'SELECT 1'
);
PREPARE stmt FROM @migrate_pnl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_pnlusd := IF(
    @has_pnlusd > 0,
    'ALTER TABLE wallet_trades DROP COLUMN pnlusd',
    'SELECT 1'
);
PREPARE stmt FROM @drop_pnlusd;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 从 dex_name 拆分版本（如 uniswap_v2）
UPDATE wallet_trades
SET dex_version = SUBSTRING_INDEX(dex_name, '_', -1),
    dex_name = SUBSTRING_INDEX(dex_name, '_', 1)
WHERE (dex_version IS NULL OR dex_version = '')
  AND dex_name LIKE '%\_v_' ESCAPE '\\';
