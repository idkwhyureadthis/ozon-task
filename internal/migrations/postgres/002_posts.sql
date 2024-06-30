-- +goose Up
CREATE TABLE posts(
    id SERIAL PRIMARY KEY,
    data TEXT NOT NULL,
    author JSON,
    is_commentable SMALLINT
);

-- +goose Down
DROP TABLE posts;
