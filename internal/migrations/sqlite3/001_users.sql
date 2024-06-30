-- +goose Up
CREATE TABLE users (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    about TEXT
);

-- +goose Down
DROP TABLE users;
