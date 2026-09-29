DROP TABLE IF EXISTS article_daily_views;

ALTER TABLE articles
    DROP COLUMN view_count,
    DROP COLUMN cover_image;
