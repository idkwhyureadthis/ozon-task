-- +goose Up
CREATE TABLE users(
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    about TEXT
);

-- +goose Down
DROP TABLE users;
