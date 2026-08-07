-- 聪明钱项目数据库表

-- 钱包评分表
CREATE TABLE IF NOT EXISTS smart_wallets (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    wallet_address VARCHAR(42) NOT NULL,
    chain_id INT NOT NULL,
    
    -- 评分维度
    score DECIMAL(10, 2) NOT NULL DEFAULT 0,
    total_pnl DECIMAL(36, 18) NOT NULL DEFAULT 0,          -- 累计盈亏（USD）
    win_rate DECIMAL(5, 2) NOT NULL DEFAULT 0,             -- 胜率（%）
    profit_loss_ratio DECIMAL(10, 2) NOT NULL DEFAULT 0,   -- 盈亏比
    max_drawdown DECIMAL(10, 2) NOT NULL DEFAULT 0,        -- 最大回撤（%）
    mainstream_ratio DECIMAL(5, 2) NOT NULL DEFAULT 0,     -- 主流币占比（%）
    trade_frequency DECIMAL(10, 2) NOT NULL DEFAULT 0,     -- 交易频率（次/天）
    
    -- 统计数据
    total_trades INT NOT NULL DEFAULT 0,
    winning_trades INT NOT NULL DEFAULT 0,
    total_volume DECIMAL(36, 18) NOT NULL DEFAULT 0,       -- 总交易量（USD）
    
    -- 评估周期
    evaluation_start_date DATETIME NOT NULL,
    evaluation_end_date DATETIME NOT NULL,
    
    -- 排名
    rank_position INT DEFAULT NULL,
    is_top_wallet BOOLEAN DEFAULT FALSE,
    
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_wallet_chain (wallet_address, chain_id),
    INDEX idx_score (score DESC),
    INDEX idx_rank (rank_position),
    INDEX idx_is_top (is_top_wallet)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 钱包交易历史表（数据来源：The Graph）
CREATE TABLE IF NOT EXISTS wallet_trades (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    wallet_address VARCHAR(42) NOT NULL,
    chain_id INT NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    block_number BIGINT UNSIGNED NOT NULL,
    block_time DATETIME NOT NULL,
    
    -- 交易信息
    dex_name VARCHAR(32) NOT NULL,                         -- uniswap
    dex_version VARCHAR(8) NOT NULL DEFAULT '',            -- v2, v3
    pool_address VARCHAR(42) NOT NULL,
    token_in VARCHAR(42) NOT NULL,
    token_out VARCHAR(42) NOT NULL,
    token_in_symbol VARCHAR(32),
    token_out_symbol VARCHAR(32),
    amount_in DECIMAL(36, 18) NOT NULL,
    amount_out DECIMAL(36, 18) NOT NULL,
    amount_usd DECIMAL(36, 2) NOT NULL,                    -- 交易金额（USD）
    
    -- 交易方向
    is_buy BOOLEAN NOT NULL,                               -- true=买入, false=卖出
    
    -- 盈亏分析（买入时为空，卖出时计算）
    pnl_usd DECIMAL(36, 18) DEFAULT NULL,
    pnl_percent DECIMAL(10, 2) DEFAULT NULL,
    holding_duration_hours INT DEFAULT NULL,
    
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_tx (tx_hash, wallet_address, token_out),
    INDEX idx_wallet (wallet_address, block_time DESC),
    INDEX idx_chain (chain_id, block_time DESC),
    INDEX idx_block_time (block_time DESC),
    INDEX idx_amount (amount_usd DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 代币信号聚合表
CREATE TABLE IF NOT EXISTS token_signals (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    token_address VARCHAR(42) NOT NULL,
    token_symbol VARCHAR(32),
    token_name VARCHAR(128),
    chain_id INT NOT NULL,
    
    -- 信号统计（最近3天）
    smart_wallet_count INT NOT NULL DEFAULT 0,             -- 买入的聪明钱包数量
    total_buy_volume DECIMAL(36, 18) NOT NULL DEFAULT 0,   -- 总买入量（USD）
    avg_buy_amount DECIMAL(36, 18) NOT NULL DEFAULT 0,     -- 平均买入金额
    first_buy_time DATETIME NOT NULL,
    last_buy_time DATETIME NOT NULL,
    
    -- 共识度评分
    consensus_score DECIMAL(10, 2) NOT NULL DEFAULT 0,     -- 综合共识度评分
    
    -- 代币基本信息
    price_usd DECIMAL(36, 18) DEFAULT NULL,
    market_cap DECIMAL(36, 2) DEFAULT NULL,
    liquidity_usd DECIMAL(36, 2) DEFAULT NULL,
    
    -- 信号周期
    signal_start_date DATETIME NOT NULL,
    signal_end_date DATETIME NOT NULL,
    
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_token_chain_period (token_address, chain_id, signal_start_date),
    INDEX idx_chain (chain_id, updated_at DESC),
    INDEX idx_consensus (consensus_score DESC),
    INDEX idx_wallet_count (smart_wallet_count DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 代币信号详情表（记录每个钱包的买入）
CREATE TABLE IF NOT EXISTS token_signal_details (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    signal_id BIGINT UNSIGNED NOT NULL,
    wallet_address VARCHAR(42) NOT NULL,
    wallet_score DECIMAL(10, 2) NOT NULL,
    
    trade_id BIGINT UNSIGNED NOT NULL,                     -- 关联 wallet_trades.id
    buy_amount_usd DECIMAL(36, 18) NOT NULL,
    buy_time DATETIME NOT NULL,
    
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (signal_id) REFERENCES token_signals(id) ON DELETE CASCADE,
    INDEX idx_signal (signal_id),
    INDEX idx_wallet (wallet_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 数据同步记录表（数据来源：The Graph）
CREATE TABLE IF NOT EXISTS sync_log (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    source VARCHAR(32) NOT NULL,                           -- thegraph
    sync_type VARCHAR(32) NOT NULL,                        -- incremental, historical, manual
    chain_id INT NOT NULL,
    
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    
    records_inserted INT NOT NULL DEFAULT 0,
    records_updated INT NOT NULL DEFAULT 0,
    records_skipped INT NOT NULL DEFAULT 0,
    
    status VARCHAR(16) NOT NULL,                           -- success, failed
    error_message TEXT,
    
    completed_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_source (source, created_at DESC),
    INDEX idx_sync_type (sync_type, created_at DESC),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
