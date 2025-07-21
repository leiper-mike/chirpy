-- +goose Up
CREATE TABLE users(
     id uuid PRIMARY KEY,
     created_at TIME,
     updated_at TIME,
     email TEXT UNIQUE NOT NULL 
);

-- +goose Down
DROP TABLE users;