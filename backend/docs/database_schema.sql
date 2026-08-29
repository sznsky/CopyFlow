-- ============================================================
-- CopyFlow 数据库完整建表 SQL
-- 数据库: MySQL 8.0 | 字符集: utf8mb4 | 引擎: InnoDB
-- 共 12 张表: 跟单模块 7 张 + 聪明钱模块 5 张
-- ============================================================

CREATE DATABASE IF NOT EXISTS copyflow
    DEFAULT CHARACTER SET utf8mb4
    DEFAULT COLLATE utf8mb4_unicode_ci;

USE copyflow;

-- ------------------------------------------------------------
-- 跟单模块
-- ------------------------------------------------------------

CREATE TABLE IF NOT EXISTS copyflow_users (
    id             BIGINT UNSIGNED  AUTO_INCREMENT PRIMARY KEY,
    wallet_address VARCHAR(42)      NULL     UNIQUE,
    email          VARCHAR(128)     NULL     UNIQUE,
    password_hash  VARCHAR(255)     NOT NULL DEFAULT '',
    nonce          VARCHAR(64)      NOT NULL DEFAULT '',
    created_at     DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copyflow_email_verifications (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    email      VARCHAR(128)    NOT NULL,
    code       VARCHAR(6)      NOT NULL,
    purpose    VARCHAR(16)     NOT NULL DEFAULT 'register',
    expires_at DATETIME(3)     NOT NULL,
    used       TINYINT(1)      NOT NULL DEFAULT 0,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_email_verifications_email (email, purpose, used)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copyflow_copy_wallets (
    id                    BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id               BIGINT UNSIGNED NOT NULL,
    chain_id              INT             NOT NULL,
    address               VARCHAR(42)     NOT NULL,
    encrypted_private_key TEXT            NOT NULL,
    is_active             TINYINT(1)      NOT NULL DEFAULT 1,
    created_at            DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_user_chain (user_id, chain_id),
    KEY idx_copy_wallets_user (user_id),
    CONSTRAINT fk_copy_wallets_user FOREIGN KEY (user_id) REFERENCES copyflow_users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copyflow_copy_configs (
    id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id        BIGINT UNSIGNED NOT NULL,
    chain_id       INT             NOT NULL,
    dex_type       VARCHAR(32)     NOT NULL,
    leader_address VARCHAR(42)     NOT NULL,
    copy_mode      VARCHAR(16)     NOT NULL DEFAULT 'ratio',
    copy_amount    DECIMAL(36,18)  NOT NULL DEFAULT 0,
    max_per_trade  DECIMAL(36,18)  NOT NULL DEFAULT 0,
    slippage_bps   INT             NOT NULL DEFAULT 300,
    is_active      TINYINT(1)      NOT NULL DEFAULT 1,
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_copy_configs_user (user_id),
    KEY idx_copy_configs_leader (chain_id, leader_address, is_active),
    CONSTRAINT fk_copy_configs_user FOREIGN KEY (user_id) REFERENCES copyflow_users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copyflow_leader_trades (
    id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    chain_id       INT             NOT NULL,
    leader_address VARCHAR(42)     NOT NULL,
    tx_hash        VARCHAR(66)     NOT NULL,
    dex_type       VARCHAR(32)     NOT NULL,
    token_in       VARCHAR(42)     NOT NULL,
    token_out      VARCHAR(42)     NOT NULL,
    amount_in      VARCHAR(78)     NOT NULL,
    amount_out     VARCHAR(78)     NOT NULL,
    block_number   BIGINT UNSIGNED NOT NULL,
    detected_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_leader_trade (chain_id, tx_hash),
    KEY idx_leader_trades_leader (chain_id, leader_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copyflow_copy_trades (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id         BIGINT UNSIGNED NOT NULL,
    config_id       BIGINT UNSIGNED NOT NULL,
    leader_trade_id BIGINT UNSIGNED NOT NULL,
    tx_hash         VARCHAR(66)     DEFAULT NULL,
    status          VARCHAR(16)     NOT NULL DEFAULT 'pending',
    amount_in       VARCHAR(78)     NOT NULL,
    token_out       VARCHAR(42)     NOT NULL,
    gas_used        BIGINT UNSIGNED DEFAULT NULL,
    error_msg       TEXT,
    created_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_copy_trades_user   (user_id),
    KEY idx_copy_trades_leader (leader_trade_id),
    CONSTRAINT fk_copy_trades_user   FOREIGN KEY (user_id)         REFERENCES copyflow_users(id)         ON DELETE CASCADE,
    CONSTRAINT fk_copy_trades_config FOREIGN KEY (config_id)       REFERENCES copyflow_copy_configs(id)  ON DELETE CASCADE,
    CONSTRAINT fk_copy_trades_leader FOREIGN KEY (leader_trade_id) REFERENCES copyflow_leader_trades(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copyflow_chain_cursors (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    chain_id   INT             NOT NULL UNIQUE,
    last_block BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ------------------------------------------------------------
-- 聪明钱模块
-- ------------------------------------------------------------

CREATE TABLE IF NOT EXISTS copyflow_smart_wallets (
    id                    BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    wallet_address        VARCHAR(42)    NOT NULL,
    chain_id              INT            NOT NULL,
    score                 DECIMAL(10,2)  NOT NULL DEFAULT 0,
    total_pnl             DECIMAL(36,18) NOT NULL DEFAULT 0,
    win_rate              DECIMAL(5,2)   NOT NULL DEFAULT 0,
    profit_loss_ratio     DECIMAL(10,2)  NOT NULL DEFAULT 0,
    max_drawdown          DECIMAL(10,2)  NOT NULL DEFAULT 0,
    mainstream_ratio      DECIMAL(5,2)   NOT NULL DEFAULT 0,
    trade_frequency       DECIMAL(10,2)  NOT NULL DEFAULT 0,
    total_trades          INT            NOT NULL DEFAULT 0,
    winning_trades        INT            NOT NULL DEFAULT 0,
    total_volume          DECIMAL(36,18) NOT NULL DEFAULT 0,
    evaluation_start_date DATETIME       NOT NULL,
    evaluation_end_date   DATETIME       NOT NULL,
    rank_position         INT            DEFAULT NULL,
    is_top_wallet         BOOLEAN        DEFAULT FALSE,
    created_at            DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_wallet_chain (wallet_address, chain_id),
    INDEX idx_score  (score DESC),
    INDEX idx_rank   (rank_position),
    INDEX idx_is_top (is_top_wallet)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copyflow_wallet_trades (
    id                     BIGINT UNSIGNED  AUTO_INCREMENT PRIMARY KEY,
    wallet_address         VARCHAR(42)      NOT NULL,
    chain_id               INT              NOT NULL,
    tx_hash                VARCHAR(66)      NOT NULL,
    block_number           BIGINT UNSIGNED  NOT NULL,
    block_time             DATETIME         NOT NULL,
    dex_name               VARCHAR(32)      NOT NULL,
    dex_version            VARCHAR(8)       NOT NULL DEFAULT '',
    pool_address           VARCHAR(42)      NOT NULL,
    token_in               VARCHAR(42)      NOT NULL,
    token_out              VARCHAR(42)      NOT NULL,
    token_in_symbol        VARCHAR(32)      DEFAULT NULL,
    token_out_symbol       VARCHAR(32)      DEFAULT NULL,
    amount_in              DECIMAL(36,18)   NOT NULL,
    amount_out             DECIMAL(36,18)   NOT NULL,
    amount_usd             DECIMAL(36,2)    NOT NULL,
    is_buy                 BOOLEAN          NOT NULL,
    pnl_usd                DECIMAL(36,18)   DEFAULT NULL,
    pnl_percent            DECIMAL(10,2)    DEFAULT NULL,
    holding_duration_hours INT              DEFAULT NULL,
    created_at             DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_tx     (tx_hash, wallet_address, token_out),
    INDEX idx_wallet     (wallet_address, block_time DESC),
    INDEX idx_chain      (chain_id, block_time DESC),
    INDEX idx_block_time (block_time DESC),
    INDEX idx_amount     (amount_usd DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copyflow_token_signals (
    id                 BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    token_address      VARCHAR(42)    NOT NULL,
    token_symbol       VARCHAR(32)    DEFAULT NULL,
    token_name         VARCHAR(128)   DEFAULT NULL,
    chain_id           INT            NOT NULL,
    smart_wallet_count INT            NOT NULL DEFAULT 0,
    total_buy_volume   DECIMAL(36,18) NOT NULL DEFAULT 0,
    avg_buy_amount     DECIMAL(36,18) NOT NULL DEFAULT 0,
    first_buy_time     DATETIME       NOT NULL,
    last_buy_time      DATETIME       NOT NULL,
    consensus_score    DECIMAL(10,2)  NOT NULL DEFAULT 0,
    price_usd          DECIMAL(36,18) DEFAULT NULL,
    market_cap         DECIMAL(36,2)  DEFAULT NULL,
    liquidity_usd      DECIMAL(36,2)  DEFAULT NULL,
    signal_start_date  DATETIME       NOT NULL,
    signal_end_date    DATETIME       NOT NULL,
    created_at         DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_token_chain_period (token_address, chain_id, signal_start_date),
    INDEX idx_chain        (chain_id, updated_at DESC),
    INDEX idx_consensus    (consensus_score DESC),
    INDEX idx_wallet_count (smart_wallet_count DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copyflow_token_signal_details (
    id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    signal_id      BIGINT UNSIGNED NOT NULL,
    wallet_address VARCHAR(42)     NOT NULL,
    wallet_score   DECIMAL(10,2)   NOT NULL,
    trade_id       BIGINT UNSIGNED NOT NULL,
    buy_amount_usd DECIMAL(36,18)  NOT NULL,
    buy_time       DATETIME        NOT NULL,
    created_at     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (signal_id) REFERENCES copyflow_token_signals(id) ON DELETE CASCADE,
    INDEX idx_signal (signal_id),
    INDEX idx_wallet (wallet_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copyflow_sync_log (
    id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    source           VARCHAR(32)    NOT NULL,
    sync_type        VARCHAR(32)    NOT NULL,
    chain_id         INT            NOT NULL,
    start_time       DATETIME       NOT NULL,
    end_time         DATETIME       NOT NULL,
    records_inserted INT            NOT NULL DEFAULT 0,
    records_updated  INT            NOT NULL DEFAULT 0,
    records_skipped  INT            NOT NULL DEFAULT 0,
    status           VARCHAR(16)    NOT NULL,
    error_message    TEXT           DEFAULT NULL,
    completed_at     DATETIME       NOT NULL,
    created_at       DATETIME       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_source    (source, created_at DESC),
    INDEX idx_sync_type (sync_type, created_at DESC),
    INDEX idx_status    (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
