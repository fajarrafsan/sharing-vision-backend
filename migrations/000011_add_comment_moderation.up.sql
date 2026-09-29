-- Komentar yang disembunyikan tetap tersimpan, hanya tidak tampil ke publik,
-- sampai admin meninjaunya.
ALTER TABLE comments
    ADD COLUMN hidden_at TIMESTAMP NULL DEFAULT NULL AFTER body,
    ADD KEY idx_comments_hidden (hidden_at);

CREATE TABLE IF NOT EXISTS comment_reports (
    comment_id BIGINT UNSIGNED NOT NULL,
    user_id    BIGINT UNSIGNED NOT NULL,
    reason     VARCHAR(20)     NOT NULL,
    created_at TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (comment_id, user_id),
    KEY idx_comment_reports_user (user_id),
    CONSTRAINT chk_comment_reports_reason CHECK (reason IN ('spam', 'abusive', 'other')),
    CONSTRAINT fk_comment_reports_comment FOREIGN KEY (comment_id) REFERENCES comments (id) ON DELETE CASCADE,
    CONSTRAINT fk_comment_reports_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
