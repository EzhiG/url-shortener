CREATE TABLE urls (
                        id SERIAL PRIMARY KEY,
                        short VARCHAR(255) NOT NULL,
                        original VARCHAR(255) NOT NULL
);

CREATE UNIQUE INDEX idx_urls_short ON urls(short);
