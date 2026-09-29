ALTER TABLE users
    ADD COLUMN email_verified_at TIMESTAMP NULL DEFAULT NULL AFTER role;

-- Akun yang sudah ada sebelum verifikasi email dianggap terverifikasi supaya
-- tidak tiba-tiba kehilangan akses berkomentar.
UPDATE users SET email_verified_at = created_at, updated_at = updated_at;

-- Token sekali pakai yang dikirim lewat email. Hanya hash-nya yang disimpan.
CREATE TABLE IF NOT EXISTS user_tokens (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id    BIGINT UNSIGNED NOT NULL,
    purpose    VARCHAR(20)     NOT NULL,
    token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    expires_at TIMESTAMP       NOT NULL,
    used_at    TIMESTAMP       NULL DEFAULT NULL,
    created_at TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_user_tokens_hash (token_hash),
    KEY idx_user_tokens_user_purpose (user_id, purpose),
    KEY idx_user_tokens_expires (expires_at),
    CONSTRAINT chk_user_tokens_purpose CHECK (purpose IN ('verify_email', 'reset_password')),
    CONSTRAINT fk_user_tokens_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
