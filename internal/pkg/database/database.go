package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/idkwhyureadthis/ozon-task/graph/model"
	"github.com/idkwhyureadthis/ozon-task/internal/pkg/auth"
	"github.com/idkwhyureadthis/ozon-task/internal/pkg/cropstrings"
	"github.com/idkwhyureadthis/ozon-task/internal/pkg/isnumber"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

type DB struct {
	Client *sql.DB
}

type Post struct {
	Id            string
	Data          string
	Author        json.RawMessage
	IsCommentable bool
}

var database *DB

func Connect(connString string, migrations string) {
	if strings.HasPrefix(connString, "postgresql://") {
		conn, err := sql.Open("postgres", connString)
		if err != nil {
			log.Fatal("unable to open postgres DB:", err)
		}
		log.Println("successfully connected to postgres DB")
		database = &DB{
			Client: conn,
		}
		database.SetupMigrations(migrations, "postgres")

	} else {
		goose.SetDialect("sqlite3")
		dbName := "internal/database/" + connString
		if connString == "" {
			dbName = "internal/database/db.sql"
		}
		if _, err := os.Stat(dbName); err != nil {
			file, err := os.Create(dbName)
			if err != nil {
				log.Fatal("failed to create db", err)
			}
			file.Close()
		}
		conn, err := sql.Open("sqlite3", dbName)
		if err != nil {
			log.Fatal("unable to open sqlite3 DB:", err)
		}
		log.Println("successfully connected to sqlite3 DB")
		database = &DB{
			Client: conn,
		}
		database.SetupMigrations(migrations, "sqlite3")
	}
}

func GetConnection() *DB {
	return database
}

func (db *DB) SetupMigrations(migrations string, drivers string) {
	pathToMigrations := "internal/migrations/" + drivers
	log.Println("setting up migrations...")
	goose.Up(db.Client, pathToMigrations)
}

func (db *DB) CreateUser(ctx context.Context, input *model.CreateUserInput) *model.User {
	var (
		name         string
		about        string
		user         model.User
		lastInsertId int
	)

	stmt, _ := db.Client.Prepare("INSERT INTO users (name, about) VALUES ($1, $2) RETURNING id;")

	name = cropstrings.CropToLength(input.Name, 32)
	if input.About != "" {
		croppedAbout := cropstrings.CropToLength(input.About, 200)
		about = croppedAbout
	}
	stmt.QueryRow(name, about).Scan(&lastInsertId)
	user = model.User{
		ID:    fmt.Sprint(lastInsertId),
		Name:  name,
		About: about,
	}
	return &user
}

func (db *DB) GetUser(ctx context.Context, id string) *model.User {
	var user model.User
	rqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := fmt.Sprintf("SELECT * FROM users WHERE id = %v;", id)
	row, err := db.Client.QueryContext(rqCtx, query)
	if err != nil {
		log.Println("failed to get user:", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.User{}
	}
	defer row.Close()
	var count int
	for row.Next() {
		err = row.Scan(&user.ID, &user.Name, &user.About)
		if err != nil {
			log.Println("failed to scan sql response", err)
			graphql.AddErrorf(ctx, "server error occurred")
			return &model.User{}
		}
		count++
	}
	if count == 0 {
		graphql.AddErrorf(ctx, "user with such id does not exist")
		return &model.User{}
	}
	if count != 1 {
		graphql.AddErrorf(ctx, "wrong user id provided")
		return &model.User{}
	}
	return &user
}

func (db *DB) GetPost(ctx context.Context, id string) *model.Post {
	var post *model.Post
	rqCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	query := fmt.Sprintf("SELECT * FROM posts WHERE id = %v", id)
	defer cancel()
	row, err := db.Client.QueryContext(rqCtx, query)
	if err != nil {
		log.Println("failed to get data from database", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Post{}
	}
	count := 0

	for row.Next() {
		var pst Post
		var author model.User
		row.Scan(&pst.Id, &pst.Data, &pst.Author, &pst.IsCommentable)
		err := json.Unmarshal(pst.Author, &author)
		if err != nil {
			log.Println("error parsing json", err)
			graphql.AddErrorf(ctx, "server error occurred")
		}
		post = &model.Post{
			ID:          pst.Id,
			Data:        pst.Data,
			Author:      &author,
			Commentable: pst.IsCommentable,
		}
		count++
	}
	if count == 0 {
		graphql.AddErrorf(ctx, "post with such id not found")
		return &model.Post{}
	}
	if count != 1 {
		graphql.AddErrorf(ctx, "wrong post id provided")
		return &model.Post{}
	}
	return post
}

func (db *DB) GetPosts(ctx context.Context, page int) []*model.Post {
	var posts []*model.Post
	limit := 20
	offset := limit * (page - 1)
	query := fmt.Sprintf("SELECT * FROM posts ORDER BY id ASC LIMIT %d OFFSET %d", limit, offset)
	if page < 1 {
		graphql.AddErrorf(ctx, "page number should be greater than 1")
		return posts
	}
	rqCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	rows, err := db.Client.QueryContext(rqCtx, query)
	if err != nil {
		log.Println("error while getting posts", err)
		graphql.AddErrorf(ctx, "error while getting posts %v", err)
		return posts
	}
	defer rows.Close()
	var cnt = 0
	for rows.Next() {
		cnt++
		var post Post
		var author model.User
		err = rows.Scan(&post.Id, &post.Data, &post.Author, &post.IsCommentable)
		if err != nil {
			graphql.AddErrorf(ctx, "error while getting posts %v", err)
			return []*model.Post{}
		}
		err = json.Unmarshal(post.Author, &author)
		if err != nil {
			log.Println("failed unmarshalling json", err)
			graphql.AddErrorf(ctx, "error while parsing json")
			return []*model.Post{}
		}
		postToAdd := model.Post{
			ID:          post.Id,
			Data:        post.Data,
			Author:      &author,
			Commentable: post.IsCommentable,
		}
		posts = append(posts, &postToAdd)
	}
	return posts
}

func (db *DB) CreatePost(ctx context.Context, input *model.CreatePostInput) *model.Post {
	var createdId int
	commentable := 0
	if input.Commentable {
		commentable = 1
	}
	userId, err := auth.IsAuthorized(ctx)
	if err != nil {
		return &model.Post{}
	}
	author := db.GetUser(ctx, userId)
	if (model.User{}) == *author {
		return &model.Post{}
	}
	authorByte, err := json.Marshal(author)
	authorJson := json.RawMessage(authorByte)
	if err != nil {
		log.Println("failed to marshall user json", err)
		graphql.AddErrorf(ctx, "failed to parse user as json")
	}
	stmt, err := db.Client.Prepare("INSERT INTO posts (data, author, is_commentable) VALUES ($1, $2, $3) returning id;")
	if err != nil {
		log.Println("error in preparing query", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Post{}
	}
	err = stmt.QueryRow(input.Data, authorJson, commentable).Scan(&createdId)
	if err != nil {
		log.Println("error in getting data", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Post{}
	}

	return &model.Post{
		ID:          fmt.Sprint(createdId),
		Data:        input.Data,
		Commentable: input.Commentable,
		Author:      author,
	}
}

func (db *DB) UpdatePost(ctx context.Context, id string, input *model.UpdatePostInput) *model.Post {
	var (
		updatedId   int
		postCreator json.RawMessage
		creatorJson model.User
		commentable = 0
	)

	if input.Commentable {
		commentable = 1
	}

	userId, err := auth.IsAuthorized(ctx)
	if err != nil {
		return &model.Post{}
	}

	author := db.GetUser(ctx, userId)
	if (model.User{}) == *author {
		return &model.Post{}
	}

	query := fmt.Sprintf("SELECT author FROM posts WHERE id = %v;", id)
	err = db.Client.QueryRow(query).Scan(&postCreator)
	if err != nil {
		log.Println("error occurred while scanning author", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Post{}
	}

	err = json.Unmarshal(postCreator, &creatorJson)

	if err != nil {
		log.Println("failed to unmarshal json", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Post{}
	}

	if creatorJson.ID != userId {
		graphql.AddErrorf(ctx, "cant change post of other users")
		return &model.Post{}
	}

	if (model.UpdatePostInput{}) == *input {
		graphql.AddErrorf(ctx, "nothing to edit")
		return &model.Post{}
	}

	query = fmt.Sprintf("UPDATE posts SET data = '%v', is_commentable = %v WHERE id = %v RETURNING id;", input.Data, commentable, id)
	err = db.Client.QueryRow(query).Scan(&updatedId)
	if err != nil {
		log.Println("error occurred while scanning postId", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Post{}
	}
	return &model.Post{
		ID:          fmt.Sprint(updatedId),
		Data:        input.Data,
		Commentable: input.Commentable,
		Author:      &creatorJson,
	}
}

func (db *DB) UpdateUser(ctx context.Context, input *model.UpdateUserInput) *model.User {
	var changedUser model.User
	userId, err := auth.IsAuthorized(ctx)
	if err != nil {
		return &model.User{}
	}
	user := db.GetUser(ctx, userId)
	if (model.User{}) == (*user) {
		return &model.User{}
	}
	query := fmt.Sprintf("UPDATE users SET about = '%v' WHERE id = %v returning *", input.About, userId)
	err = db.Client.QueryRow(query).Scan(&changedUser.ID, &changedUser.Name, &changedUser.About)
	if err != nil {
		log.Println("error scanning data", err)
		graphql.AddErrorf(ctx, "error parsing data")
		return &model.User{}
	}
	return &changedUser
}

func (db *DB) CreateComment(ctx context.Context, input *model.CreateCommentInput) *model.Comment {
	text := input.Text
	if len(input.Text) > 2000 {
		text = cropstrings.CropToLength(text, 2000)
	}
	userId, err := auth.IsAuthorized(ctx)
	initialComment := -1
	answerTo := isnumber.TryConvertToInt(input.AnswerTo)
	if err != nil {
		return &model.Comment{}
	}
	user := db.GetUser(ctx, userId)
	if (model.User{}) == (*user) {
		return &model.Comment{}
	}
	userJson, err := json.Marshal(user)
	if err != nil {
		log.Println("error marshalling JSON", err)
		graphql.AddErrorf(ctx, "internal server error")
		return &model.Comment{}
	}
	post := db.GetPost(ctx, input.Post)
	if (model.Post{}) == (*post) {
		return &model.Comment{}
	}
	if !post.Commentable {
		graphql.AddErrorf(ctx, "cannot comment this post (commenting disabled)")
		return &model.Comment{}
	}
	postJson, err := json.Marshal(post)
	if err != nil {
		log.Println("error marshalling JSON", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Comment{}
	}
	if answerTo != -1 {
		var initialCommentStr string
		updateAnswerQuery := fmt.Sprintf("UPDATE comments SET has_replies = 1 WHERE id = %d RETURNING answer_to", answerTo)
		err = db.Client.QueryRow(updateAnswerQuery).Scan(&initialCommentStr)
		if err != nil {
			graphql.AddErrorf(ctx, "failed to answer to comment that doesn't exist")
			return &model.Comment{}
		}
		initialComment = isnumber.TryConvertToInt(initialCommentStr)
	}
	query := fmt.Sprintf(`
	INSERT INTO COMMENTS (post, author, initial_comment, answer_to, data, has_replies)
	VALUES ('%s', '%s', %v, %d, '%s', %d) RETURNING id;`, postJson, userJson, initialComment, answerTo, text, 0)
	var createdID int
	err = db.Client.QueryRow(query).Scan(&createdID)
	if err != nil {
		log.Println("error inserting comment", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Comment{}
	}
	return &model.Comment{
		ID:             fmt.Sprint(createdID),
		Text:           text,
		Post:           post,
		Creator:        user,
		InitialComment: fmt.Sprint(initialComment),
		AnswerTo:       fmt.Sprint(answerTo),
	}
}

func (db *DB) GetComment(ctx context.Context, id string) *model.Comment {
	type DBResponse struct {
		id              int
		post            json.RawMessage
		author          json.RawMessage
		initial_comment int
		answer_to       int
		data            string
		has_replies     int
	}
	var resp DBResponse
	var comm model.Comment

	query := fmt.Sprintf(`SELECT * FROM comments WHERE id = %v`, id)

	err := db.Client.QueryRow(query).Scan(&resp.id, &resp.post, &resp.author, &resp.initial_comment, &resp.answer_to, &resp.data, &resp.has_replies)
	if err == sql.ErrNoRows {
		graphql.AddErrorf(ctx, "comment with such id does not exits")
		return &model.Comment{}
	}
	if err != nil {
		log.Println("failed to get data from DB", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Comment{}
	}
	err = json.Unmarshal(resp.post, &comm.Post)
	if err != nil {
		log.Println("failed to marshall json", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Comment{}
	}
	err = json.Unmarshal(resp.author, &comm.Creator)
	if err != nil {
		log.Println("failed to marshall json", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Comment{}
	}
	comm.InitialComment = fmt.Sprint(resp.initial_comment)
	comm.ID = fmt.Sprint(resp.id)
	comm.AnswerTo = fmt.Sprint(resp.answer_to)
	comm.Text = resp.data

	return &comm
}

func (db *DB) UpdateComment(ctx context.Context, commId string, input *model.UpdateCommentInput) *model.Comment {
	type DBResponse struct {
		id             int
		post           json.RawMessage
		author         json.RawMessage
		initialComment int
		data           string
		answerTo       string
		hasReplies     int
	}
	userId, err := auth.IsAuthorized(ctx)
	if err != nil {
		return &model.Comment{}
	}

	user := db.GetUser(ctx, userId)
	if (*user) == (model.User{}) {
		return &model.Comment{}
	}

	var authorJson json.RawMessage
	var author model.User

	getAuthorQuery := fmt.Sprintf(`SELECT author FROM comments WHERE id = %s`, commId)

	err = db.Client.QueryRow(getAuthorQuery).Scan(&authorJson)

	if err == sql.ErrNoRows {
		graphql.AddErrorf(ctx, "comment with such id not found")
		return &model.Comment{}
	}
	if err != nil {
		log.Println("error occurred scanning DB", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Comment{}
	}

	err = json.Unmarshal(authorJson, &author)

	if err != nil {
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Comment{}
	}

	if author.ID != userId {
		graphql.AddErrorf(ctx, "can't edit comment of other person")
		return &model.Comment{}
	}

	query := fmt.Sprintf(`UPDATE comments SET data = '%s' WHERE id = %s returning *;`, input.Data, commId)

	resp := DBResponse{}
	err = db.Client.QueryRow(query).Scan(&resp.id, &resp.post, &resp.author, &resp.initialComment, &resp.answerTo, &resp.data, &resp.hasReplies)

	if err != nil {
		log.Println("failed to parse data from query", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Comment{}
	}

	commentPost := model.Post{}
	err = json.Unmarshal(resp.post, &commentPost)
	if err != nil {
		log.Println("failed to unmarshal data", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return &model.Comment{}
	}
	newComment := model.Comment{
		ID:             fmt.Sprint(resp.id),
		Text:           input.Data,
		Post:           &commentPost,
		AnswerTo:       fmt.Sprint(resp.answerTo),
		InitialComment: fmt.Sprint(resp.initialComment),
		Creator:        &author,
	}
	return &newComment
}

func (db *DB) GetComments(ctx context.Context, postID string, page int) []*model.Comment {
	limit := 20
	offset := (page - 1) * limit
	if page < 1 {
		graphql.AddErrorf(ctx, "pages start with 1")
		return []*model.Comment{}
	}
	post := db.GetPost(ctx, postID)
	if post == (&model.Post{}) {
		graphql.AddErrorf(ctx, "post not found")
		return []*model.Comment{}
	}
	postJson, err := json.Marshal(post)
	if err != nil {
		log.Println("failed to marshal json", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return []*model.Comment{}
	}
	query := fmt.Sprintf(`SELECT * FROM comments WHERE post::text = '%s' AND answer_to = -1 LIMIT %d OFFSET %d`, postJson, limit, offset)

	rows, err := db.Client.Query(query)
	if err != nil {
		log.Println("error performing query", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return []*model.Comment{}
	}

	comments := []*model.Comment{}

	type DBResponse struct {
		id             int
		post           json.RawMessage
		author         json.RawMessage
		initialComment int
		answerTo       int
		data           string
		hasReplies     int
	}
	for rows.Next() {
		resp := DBResponse{}
		err = rows.Scan(&resp.id, &resp.post, &resp.author, &resp.initialComment, &resp.answerTo, &resp.data, &resp.hasReplies)
		hasReplies := false
		if resp.hasReplies > 0 {
			hasReplies = true
		}
		post := model.Post{}
		err = json.Unmarshal(resp.post, &post)
		if err != nil {
			log.Println("error decoding json", err)
			graphql.AddErrorf(ctx, "server error occurred")
			return []*model.Comment{}
		}

		creator := model.User{}
		err = json.Unmarshal(resp.author, &creator)
		if err != nil {
			log.Println("error decoding json", err)
			graphql.AddErrorf(ctx, "server error occurred")
			return []*model.Comment{}
		}

		comment := model.Comment{
			ID:             fmt.Sprint(resp.id),
			Text:           resp.data,
			Post:           &post,
			AnswerTo:       fmt.Sprint(resp.answerTo),
			InitialComment: fmt.Sprint(resp.initialComment),
			Creator:        &creator,
			HasReplies:     hasReplies,
		}
		comments = append(comments, &comment)
	}
	return comments
}

func (db *DB) GetReplies(ctx context.Context, commentId string, page int) []*model.Comment {
	limit := 20
	offset := (page - 1) * limit
	if page < 1 {
		graphql.AddErrorf(ctx, "pages start with 1")
		return []*model.Comment{}
	}
	commentIdInt := isnumber.TryConvertToInt(commentId)
	if commentIdInt == -1 {
		graphql.AddErrorf(ctx, "wrong commentId provided")
		return []*model.Comment{}
	}
	if db.GetComment(ctx, commentId) == (&model.Comment{}) {
		return []*model.Comment{}
	}

	query := fmt.Sprintf("SELECT * FROM comments WHERE answer_to = %d LIMIT %d OFFSET %d", commentIdInt, limit, offset)

	rows, err := db.Client.Query(query)

	if err != nil {
		log.Println("failed to perform query", err)
		graphql.AddErrorf(ctx, "server error occurred")
		return []*model.Comment{}
	}

	comments := []*model.Comment{}

	type DBResponse struct {
		id             int
		post           json.RawMessage
		author         json.RawMessage
		initialComment int
		answerTo       int
		data           string
		hasReplies     int
	}
	for rows.Next() {
		resp := DBResponse{}
		err = rows.Scan(&resp.id, &resp.post, &resp.author, &resp.initialComment, &resp.answerTo, &resp.data, &resp.hasReplies)
		hasReplies := false
		if resp.hasReplies > 0 {
			hasReplies = true
		}
		post := model.Post{}
		err = json.Unmarshal(resp.post, &post)
		if err != nil {
			log.Println("error decoding json", err)
			graphql.AddErrorf(ctx, "server error occurred")
			return []*model.Comment{}
		}

		creator := model.User{}
		err = json.Unmarshal(resp.author, &creator)
		if err != nil {
			log.Println("error decoding json", err)
			graphql.AddErrorf(ctx, "server error occurred")
			return []*model.Comment{}
		}

		comment := model.Comment{
			ID:             fmt.Sprint(resp.id),
			Text:           resp.data,
			Post:           &post,
			AnswerTo:       fmt.Sprint(resp.answerTo),
			InitialComment: fmt.Sprint(resp.initialComment),
			Creator:        &creator,
			HasReplies:     hasReplies,
		}
		comments = append(comments, &comment)
	}
	return comments
}
