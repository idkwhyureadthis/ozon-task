-- +goose Up
CREATE TABLE comments(
    id SERIAL PRIMARY KEY,
    post JSON,
    author JSON,
    initial_comment SERIAL,
    answer_to SERIAL,
    data TEXT NOT NULL,
    has_replies SMALLINT
);



-- +goose Down
DROP TABLE comments;