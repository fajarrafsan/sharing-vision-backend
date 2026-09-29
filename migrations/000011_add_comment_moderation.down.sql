DROP TABLE IF EXISTS comment_reports;

ALTER TABLE comments
    DROP INDEX idx_comments_hidden,
    DROP COLUMN hidden_at;
