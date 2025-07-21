-- +goose up
CREATE TABLE chirps(
     id uuid PRIMARY KEY,
     created_at TIME,
     updated_at TIME,
     body TEXT NOT NULL,
     user_id uuid NOT NULL 
     REFERENCES users
     ON DELETE CASCADE
);
-- +goose down
DROP TABLE chirps;