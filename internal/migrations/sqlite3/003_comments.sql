-- +goose Up
CREATE TABLE comments (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	post TEXT,
	author TEXT,
	initial_comment INTEGER,
	asnwer_to INTEGER,
	data TEXT NOT NULL,
	has_replies INTEGER
);



-- +goose Down
DROP TABLE comments;