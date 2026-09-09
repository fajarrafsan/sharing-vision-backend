-- Pembuatan database dan tabel posts secara manual, sebagai alternatif migrate.
--   mysql -u root -p < docs/schema.sql

CREATE DATABASE IF NOT EXISTS article
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE article;

CREATE TABLE IF NOT EXISTS posts (
    id           INT          NOT NULL AUTO_INCREMENT,
    title        VARCHAR(200) NOT NULL,
    content      TEXT         NOT NULL,
    category     VARCHAR(100) NOT NULL,
    created_date TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_date TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    status       VARCHAR(100) NOT NULL COMMENT 'publish | draft | thrash',
    PRIMARY KEY (id),
    KEY idx_posts_status (status),
    KEY idx_posts_category (category),
    KEY idx_posts_created_date (created_date)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;
