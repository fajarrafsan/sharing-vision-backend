-- Mengembalikan articles menjadi posts seperti versi awal. Penulis, slug, dan
-- waktu terbit tidak punya tempat di skema lama sehingga ikut hilang.

ALTER TABLE articles
    DROP FOREIGN KEY fk_articles_author,
    DROP FOREIGN KEY fk_articles_category,
    DROP CHECK chk_articles_status;

ALTER TABLE articles
    ADD COLUMN category VARCHAR(100) NOT NULL DEFAULT '' AFTER content;

UPDATE articles a
JOIN categories c ON c.id = a.category_id
SET a.category = c.name, a.updated_at = a.updated_at;

UPDATE articles
SET status = CASE status
        WHEN 'published' THEN 'publish'
        WHEN 'archived' THEN 'thrash'
        ELSE 'draft'
    END,
    updated_at = updated_at;

ALTER TABLE articles
    DROP INDEX uq_articles_slug,
    DROP INDEX idx_articles_author,
    DROP INDEX idx_articles_category,
    DROP INDEX idx_articles_status_published,
    DROP INDEX idx_articles_created,
    DROP COLUMN author_id,
    DROP COLUMN category_id,
    DROP COLUMN slug,
    DROP COLUMN published_at,
    MODIFY id INT NOT NULL AUTO_INCREMENT,
    MODIFY content TEXT NOT NULL,
    MODIFY category VARCHAR(100) NOT NULL,
    MODIFY status VARCHAR(100) NOT NULL COMMENT 'publish | draft | thrash',
    RENAME COLUMN created_at TO created_date,
    RENAME COLUMN updated_at TO updated_date,
    ADD KEY idx_posts_status (status),
    ADD KEY idx_posts_category (category),
    ADD KEY idx_posts_created_date (created_date);

RENAME TABLE articles TO posts;

DELETE FROM users WHERE email = 'arsip@warta.local';
