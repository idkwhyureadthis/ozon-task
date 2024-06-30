// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

type Comment struct {
	ID             string `json:"id"`
	Text           string `json:"text"`
	Post           *Post  `json:"post"`
	AnswerTo       string `json:"answer_to"`
	InitialComment string `json:"initial_comment"`
	Creator        *User  `json:"creator"`
	HasReplies     bool   `json:"hasReplies"`
}

type CreateCommentInput struct {
	Text     string `json:"text"`
	Post     string `json:"post"`
	AnswerTo string `json:"answer_to"`
}

type CreatePostInput struct {
	Data        string `json:"data"`
	Commentable bool   `json:"commentable"`
}

type CreateUserInput struct {
	Name  string `json:"name"`
	About string `json:"about"`
}

type Mutation struct {
}

type Post struct {
	ID          string `json:"id"`
	Data        string `json:"data"`
	Commentable bool   `json:"commentable"`
	Author      *User  `json:"author"`
}

type Query struct {
}

type UpdateCommentInput struct {
	Data string `json:"data"`
}

type UpdatePostInput struct {
	Data        string `json:"data"`
	Commentable bool   `json:"commentable"`
}

type UpdateUserInput struct {
	About string `json:"about"`
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	About string `json:"about"`
}
