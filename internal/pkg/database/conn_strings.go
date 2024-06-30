package database

var createUserString = `CREATE TABLE users (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "name" TEXT NOT NULL,
    "about" TEXT
	);`

var createPostsString = ``

var createCommentsString = `CREATE TABLE comments (
	"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	"post" TEXT,
	"author" TEXT,
	"initial_comment" INTEGER,
	"asnwer_to" INTEGER,
	"data TEXT" NOT NULL,
	"has_replies" INTEGER
	);`

func getSetupString() []string {
	return []string{createUserString, createPostsString, createCommentsString}
}
