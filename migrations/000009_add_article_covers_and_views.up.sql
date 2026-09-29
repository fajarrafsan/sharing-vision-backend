ALTER TABLE articles
    ADD COLUMN cover_image VARCHAR(255) NULL DEFAULT NULL AFTER content,
    ADD COLUMN view_count INT UNSIGNED NOT NULL DEFAULT 0 AFTER status;

-- Jumlah dibaca per artikel per hari, bahan grafik di dashboard.
CREATE TABLE IF NOT EXISTS article_daily_views (
    article_id BIGINT UNSIGNED NOT NULL,
    day        DATE            NOT NULL,
    views      INT UNSIGNED    NOT NULL DEFAULT 0,
    PRIMARY KEY (article_id, day),
    KEY idx_article_daily_views_day (day),
    CONSTRAINT fk_article_daily_views_article FOREIGN KEY (article_id) REFERENCES articles (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
