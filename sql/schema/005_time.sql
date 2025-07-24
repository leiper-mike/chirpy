-- +goose up
ALTER TABLE users
DROP created_at,
DROP updated_at,
ADD created_at TIMESTAMP,
ADD updated_at TIMESTAMP;
ALTER TABLE chirps
DROP created_at,
DROP updated_at,
ADD created_at TIMESTAMP,
ADD updated_at TIMESTAMP;
ALTER TABLE refresh_tokens
DROP created_at,
DROP updated_at,
DROP expires_at,
DROP revoked_at,
ADD created_at TIMESTAMP,
ADD updated_at TIMESTAMP,
ADD expires_at TIMESTAMP NOT NULL,
ADD revoked_at TIMESTAMP;
-- +goose down
ALTER TABLE users
ALTER created_at TYPE TIME,
ALTER updated_at TYPE TIME;
ALTER TABLE chirps
ALTER created_at TYPE TIME,
ALTER updated_at TYPE TIME;
ALTER TABLE refresh_tokens
ALTER created_at TYPE TIME,
ALTER updated_at TYPE TIME,
ALTER expires_at TYPE TIME;