CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    wallet_address VARCHAR(42) NOT NULL UNIQUE,
    nonce VARCHAR(64) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copy_wallets (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    chain_id INT NOT NULL,
    address VARCHAR(42) NOT NULL,
    encrypted_private_key TEXT NOT NULL,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_user_chain (user_id, chain_id),
    KEY idx_copy_wallets_user (user_id),
    CONSTRAINT fk_copy_wallets_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copy_configs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    chain_id INT NOT NULL,
    dex_type VARCHAR(32) NOT NULL,
    leader_address VARCHAR(42) NOT NULL,
    copy_mode VARCHAR(16) NOT NULL DEFAULT 'ratio',
    copy_amount DECIMAL(36,18) NOT NULL DEFAULT 0,
    max_per_trade DECIMAL(36,18) NOT NULL DEFAULT 0,
    slippage_bps INT NOT NULL DEFAULT 300,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_copy_configs_user (user_id),
    KEY idx_copy_configs_leader (chain_id, leader_address, is_active),
    CONSTRAINT fk_copy_configs_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS leader_trades (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    chain_id INT NOT NULL,
    leader_address VARCHAR(42) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    dex_type VARCHAR(32) NOT NULL,
    token_in VARCHAR(42) NOT NULL,
    token_out VARCHAR(42) NOT NULL,
    amount_in VARCHAR(78) NOT NULL,
    amount_out VARCHAR(78) NOT NULL,
    block_number BIGINT UNSIGNED NOT NULL,
    detected_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_leader_trade (chain_id, tx_hash),
    KEY idx_leader_trades_leader (chain_id, leader_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copy_trades (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    config_id BIGINT UNSIGNED NOT NULL,
    leader_trade_id BIGINT UNSIGNED NOT NULL,
    tx_hash VARCHAR(66) DEFAULT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    amount_in VARCHAR(78) NOT NULL,
    token_out VARCHAR(42) NOT NULL,
    gas_used BIGINT UNSIGNED DEFAULT NULL,
    error_msg TEXT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_copy_trades_user (user_id),
    KEY idx_copy_trades_leader (leader_trade_id),
    CONSTRAINT fk_copy_trades_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_copy_trades_config FOREIGN KEY (config_id) REFERENCES copy_configs(id) ON DELETE CASCADE,
    CONSTRAINT fk_copy_trades_leader FOREIGN KEY (leader_trade_id) REFERENCES leader_trades(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Worker cursor for block scanning (extensible per chain)
CREATE TABLE IF NOT EXISTS chain_cursors (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    chain_id INT NOT NULL UNIQUE,
    last_block BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
