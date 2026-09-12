DROP INDEX IF EXISTS idx_urls_original_user;
CREATE UNIQUE INDEX idx_urls_original ON urls (original);
