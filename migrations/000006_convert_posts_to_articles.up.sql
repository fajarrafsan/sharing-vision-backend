-- Mengubah tabel posts dari versi awal menjadi articles tanpa membuang data.

-- Post lama belum punya penulis. Semuanya diberikan ke akun arsip yang tidak
-- bisa dipakai login karena password_hash-nya bukan hash bcrypt.
INSERT INTO users (name, email, password_hash, role)
SELECT 'Arsip Warta', 'arsip@warta.local', '!', 'author'
FROM DUAL
WHERE EXISTS (SELECT 1 FROM posts);

-- Kategori yang tadinya teks bebas dipindah ke tabel categories. Nama yang
-- menghasilkan slug sama diberi akhiran -2, -3, dan seterusnya.
INSERT INTO categories (name, slug)
SELECT name, IF(rn = 1, base, CONCAT(base, '-', rn))
FROM (
    SELECT name, base, ROW_NUMBER() OVER (PARTITION BY base ORDER BY name) AS rn
    FROM (
        SELECT name,
               CASE
                   WHEN s = '' THEN 'kategori'
                   WHEN s REGEXP '^[0-9]+$' THEN CONCAT('kategori-', s)
                   ELSE s
               END AS base
        FROM (
            SELECT DISTINCT TRIM(category) AS name,
                   TRIM(BOTH '-' FROM REGEXP_REPLACE(LOWER(TRIM(category)), '[^a-z0-9]+', '-')) AS s
            FROM posts
        ) AS raw_categories
    ) AS based
) AS numbered;

ALTER TABLE posts
    MODIFY id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ADD COLUMN author_id BIGINT UNSIGNED NULL AFTER id,
    ADD COLUMN category_id BIGINT UNSIGNED NULL AFTER author_id,
    ADD COLUMN slug VARCHAR(220) NULL AFTER title,
    ADD COLUMN published_at TIMESTAMP NULL DEFAULT NULL AFTER status;

-- updated_date = updated_date mencegah ON UPDATE CURRENT_TIMESTAMP menimpa
-- waktu ubah asli setiap post.
UPDATE posts p
JOIN categories c ON c.name = TRIM(p.category)
SET p.category_id = c.id, p.updated_date = p.updated_date;

UPDATE posts
SET author_id = (SELECT id FROM users WHERE email = 'arsip@warta.local'),
    slug = CONCAT(
        COALESCE(NULLIF(TRIM(BOTH '-' FROM REGEXP_REPLACE(LOWER(title), '[^a-z0-9]+', '-')), ''), 'artikel'),
        '-', id),
    status = CASE LOWER(status)
        WHEN 'publish' THEN 'published'
        WHEN 'thrash' THEN 'archived'
        ELSE 'draft'
    END,
    updated_date = updated_date;

UPDATE posts
SET published_at = created_date, updated_date = updated_date
WHERE status = 'published';

ALTER TABLE posts
    MODIFY author_id BIGINT UNSIGNED NOT NULL,
    MODIFY category_id BIGINT UNSIGNED NOT NULL,
    MODIFY slug VARCHAR(220) NOT NULL,
    MODIFY content MEDIUMTEXT NOT NULL,
    MODIFY status VARCHAR(20) NOT NULL DEFAULT 'draft',
    DROP INDEX idx_posts_status,
    DROP INDEX idx_posts_category,
    DROP INDEX idx_posts_created_date,
    DROP COLUMN category,
    RENAME COLUMN created_date TO created_at,
    RENAME COLUMN updated_date TO updated_at;

RENAME TABLE posts TO articles;

ALTER TABLE articles
    ADD UNIQUE KEY uq_articles_slug (slug),
    ADD KEY idx_articles_author (author_id),
    ADD KEY idx_articles_category (category_id),
    ADD KEY idx_articles_status_published (status, published_at),
    ADD KEY idx_articles_created (created_at),
    ADD CONSTRAINT chk_articles_status CHECK (status IN ('draft', 'published', 'archived')),
    ADD CONSTRAINT fk_articles_author FOREIGN KEY (author_id) REFERENCES users (id),
    ADD CONSTRAINT fk_articles_category FOREIGN KEY (category_id) REFERENCES categories (id);
