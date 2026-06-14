-- 邮箱注册登录扩展（在 001 之后执行）
ALTER TABLE users
    MODIFY wallet_address VARCHAR(42) NULL,
    ADD COLUMN email VARCHAR(128) NULL UNIQUE AFTER wallet_address,
    ADD COLUMN password_hash VARCHAR(255) NOT NULL DEFAULT '' AFTER email;

CREATE TABLE IF NOT EXISTS email_verifications (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(128) NOT NULL,
    code VARCHAR(6) NOT NULL,
    purpose VARCHAR(16) NOT NULL DEFAULT 'register',
    expires_at DATETIME(3) NOT NULL,
    used TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_email_verifications_email (email, purpose, used)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
