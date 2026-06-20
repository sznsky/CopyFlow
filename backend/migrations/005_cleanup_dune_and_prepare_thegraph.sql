-- 清理旧 Dune 数据并优化表结构

-- 临时禁用外键检查
SET FOREIGN_KEY_CHECKS = 0;

-- 1. 删除所有 wallet_trades 数据（准备从 The Graph 重新拉取）
TRUNCATE TABLE wallet_trades;

-- 2. 清理 smart_wallets 评分数据（重新计算）
TRUNCATE TABLE smart_wallets;

-- 3. 清理 token_signals 和 token_signal_details
TRUNCATE TABLE token_signal_details;
TRUNCATE TABLE token_signals;

-- 恢复外键检查
SET FOREIGN_KEY_CHECKS = 1;

-- 4. 重命名 dune_sync_log 为 sync_log（如果存在）
DROP TABLE IF EXISTS sync_log;
CREATE TABLE sync_log (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    source VARCHAR(32) NOT NULL,                          -- thegraph
    sync_type VARCHAR(32) NOT NULL,                       -- incremental, historical, manual
    chain_id INT NOT NULL,
    
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    
    records_inserted INT NOT NULL DEFAULT 0,
    records_updated INT NOT NULL DEFAULT 0,
    records_skipped INT NOT NULL DEFAULT 0,
    
    status VARCHAR(16) NOT NULL,                          -- success, failed
    error_message TEXT,
    
    completed_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_source (source, created_at DESC),
    INDEX idx_sync_type (sync_type, created_at DESC),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 5. 删除旧的 dune_sync_log 表
DROP TABLE IF EXISTS dune_sync_log;
