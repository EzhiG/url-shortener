DROP INDEX IF EXISTS idx_urls_original;
CREATE UNIQUE INDEX idx_urls_original_user ON urls(original, user_id);
