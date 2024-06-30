-- +goose Up
CREATE TABLE posts (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	data TEXT NOT NULL,
	author TEXT,
	is_commentable INTEGER
);

-- +goose Down
DROP TABLE posts;
