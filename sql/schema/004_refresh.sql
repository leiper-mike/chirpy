-- +goose up
CREATE TABLE refresh_tokens(
     token TEXT PRIMARY KEY,
     created_at TIME,
     updated_at TIME,
     user_id uuid NOT NULL 
     REFERENCES users
     ON DELETE CASCADE,
     expires_at TIME NOT NULL,
     revoked_at TIME
);
-- +goose down
DROP TABLE refresh_tokens;