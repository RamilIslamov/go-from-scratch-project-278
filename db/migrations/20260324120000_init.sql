-- +goose Up
CREATE TABLE IF NOT EXISTS app_init (
                                        id BIGSERIAL PRIMARY KEY,
                                        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

-- +goose Down
DROP TABLE IF EXISTS app_init;