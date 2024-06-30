package graph

import (
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/idkwhyureadthis/ozon-task/internal/pkg/database"
	"github.com/idkwhyureadthis/ozon-task/internal/pkg/mw"
	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	database.Connect("postgresql://postgres:12345@localhost:5432/ozon-task?sslmode=disable", "RESET")
	database.GetConnection().Client.Exec(`TRUNCATE users RESTART IDENTITY;`)
	database.GetConnection().Client.Exec(`TRUNCATE posts RESTART IDENTITY;`)
	database.GetConnection().Client.Exec(`TRUNCATE comments RESTART IDENTITY;`)
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	c := client.New(mw.AuthMiddleware(srv))
	Init()

	t.Run("create 2 users and update text of the first one", func(t *testing.T) {
		var resp struct {
			CreateUser struct{ ID string }
		}
		c.MustPost(`mutation{createUser(input:{name:"srgold78" about:"Влад Младший"}){id}}`, &resp)

		require.Equal(t, resp.CreateUser.ID, "1")

		c.MustPost(`mutation{createUser(input:{name:"srgold77" about:"Влад Старший"}){id}}`, &resp)

		require.Equal(t, resp.CreateUser.ID, "2")
	})

	t.Run("update data of the first user and then get it", func(t *testing.T) {
		type UpdateResp struct {
			UpdateUser struct {
				About string
			}
		}

		resp1 := UpdateResp{}

		err := c.Post(`mutation{updateUser(input:{about:"я srgold77"}){about}}`, &resp1)

		require.EqualError(t, err, `[{"message":"not authorized","path":["updateUser"]}]`)

		resp2 := UpdateResp{}
		c.MustPost(`mutation{updateUser(input:{about:"я srgold77"}){about}}`, &resp2, client.AddHeader("user", "2"))
		require.Equal(t, resp2.UpdateUser.About, "я srgold77")

		type GetResp struct {
			Get_user struct {
				ID    string
				Name  string
				About string
			}
		}

		firstResp := GetResp{}
		c.MustPost(`query{get_user(id:1) {id name about}}`, &firstResp)

		require.Equal(t, firstResp.Get_user.Name, "srgold78")

		secondResp := GetResp{}
		c.MustPost(`query{get_user(id:2) {id name about}}`, &secondResp)

		require.Equal(t, secondResp.Get_user.About, "я srgold77")
	})

	t.Run("create new post, then get it, edit and again get it", func(t *testing.T) {
		type Resps struct {
			CreatePost struct {
				ID          string
				Data        string
				Commentable bool
			}
			Get_post struct {
				ID          string
				Data        string
				Commentable bool
			}
			UpdatePost struct {
				ID          string
				Data        string
				Commentable bool
			}
		}
		resps := Resps{}
		err := c.Post(`mutation{createPost(input:{data:"это пост srgold77" commentable:false}){id data commentable}}`, &resps)
		require.EqualError(t, err, `[{"message":"not authorized","path":["createPost"]}]`)

		c.MustPost(`mutation{createPost(input:{data:"это пост srgold77" commentable:false}){id data commentable}}`, &resps, client.AddHeader("user", "2"))
		require.Equal(t, resps.CreatePost.ID, "1")
		require.Equal(t, resps.CreatePost.Data, "это пост srgold77")
		require.Equal(t, resps.CreatePost.Commentable, false)

		c.MustPost(`query{get_post(post_id:"1") {id data commentable}}`, &resps)
		require.Equal(t, resps.CreatePost, resps.Get_post)

		err = c.Post(`mutation{updatePost(id:1 input:{data:"это изменённый пост srgold77" commentable:true}){commentable data}}`, &resps, client.AddHeader("user", "-1"))
		require.EqualError(t, err, `[{"message":"wrong user id","path":["updatePost"]}]`)

		err = c.Post(`mutation{updatePost(id:1 input:{data:"это изменённый пост srgold77" commentable:true}){commentable data}}`, &resps, client.AddHeader("user", "1"))
		require.EqualError(t, err, `[{"message":"cant change post of other users","path":["updatePost"]}]`)

		err = c.Post(`mutation{updatePost(id:1 input:{data:"это изменённый пост srgold77" commentable:true}){commentable data}}`, &resps)
		require.EqualError(t, err, `[{"message":"not authorized","path":["updatePost"]}]`)

		c.MustPost(`mutation{updatePost(id:1 input:{data:"это изменённый пост srgold77" commentable:true}){commentable data}}`, &resps, client.AddHeader("user", "2"))
		require.Equal(t, resps.UpdatePost.Data, "это изменённый пост srgold77")
		require.Equal(t, resps.UpdatePost.Commentable, true)
	})

	t.Run("create 30 comments and get first and second page", func(t *testing.T) {
		type CommentPage struct {
			Posts []struct {
				ID string
			}
		}
		firstPage := CommentPage{}
		secondPage := CommentPage{}
		database.GetConnection().Client.Exec(`TRUNCATE posts RESTART IDENTITY;`)
		for i := range 30 {
			data := fmt.Sprintf("Это пост номер %d", i+1)
			_, err := c.RawPost(fmt.Sprintf(`mutation{createPost(input:{data:"%s" commentable: true}){id}}`, data), client.AddHeader("user", "2"))
			if err != nil {
				t.Error("Failed creating comment", err)
			}
		}
		c.MustPost(`query{posts(page:1){id}}`, &firstPage)
		c.MustPost(`query{posts(page:2){id}}`, &secondPage)

		require.Equal(t, len(firstPage.Posts), 20)
		require.Equal(t, len(secondPage.Posts), 10)
	})

	t.Run("create comment to 21st post and get it", func(t *testing.T) {
		var resp struct {
			CreateComment struct {
				Text    string
				Creator struct {
					ID    string
					Name  string
					About string
				}
			}
			Get_comment struct {
				Text    string
				Creator struct {
					ID    string
					Name  string
					About string
				}
			}
		}
		err := c.Post(`mutation{createComment(input:{text:"это комментарий к 21 посту", post: 21, answer_to: -1}){text creator{id name about}}}`, &resp)
		require.EqualError(t, err, `[{"message":"not authorized","path":["createComment"]},{"message":"the requested element is null which the schema does not allow","path":["createComment","creator"]}]`)
		err = c.Post(`mutation{createComment(input:{text:"это комментарий к 21 посту", post: 21, answer_to: -1}){text creator{id name about}}}`, &resp, client.AddHeader("user", "-1"))
		require.EqualError(t, err, `[{"message":"wrong user id","path":["createComment"]},{"message":"the requested element is null which the schema does not allow","path":["createComment","creator"]}]`)
		c.MustPost(`mutation{createComment(input:{text:"это комментарий к 21 посту от srgold78", post: 21, answer_to: -1}){text creator{id name about}}}`, &resp, client.AddHeader("user", "1"))
		require.Equal(t, resp.CreateComment.Text, "это комментарий к 21 посту от srgold78")
		require.Equal(t, resp.CreateComment.Creator.ID, "1")
		require.Equal(t, resp.CreateComment.Creator.Name, "srgold78")
		c.MustPost(`query{get_comment(comment_id: 1){text creator{id name about}}}`, &resp)
		require.Equal(t, resp.CreateComment, resp.Get_comment)
	})

	t.Run("update created comment and get check if it changed", func(t *testing.T) {
		var resp struct {
			UpdateComment struct {
				Text    string
				ID      string
				Creator struct {
					ID    string
					Name  string
					About string
				}
			}
			Get_comment struct {
				Text    string
				ID      string
				Creator struct {
					ID    string
					Name  string
					About string
				}
			}
		}
		err := c.Post(`mutation{updateComment(comm_id:"1" input:{data:"это изменённый комментарий"}){text id creator{id name about}}}`, &resp, client.AddHeader("user", "-1"))
		require.Error(t, err, `[{"message":"wrong user id","path":["updateComment"]},{"message":"the requested element is null which the schema does not allow","path":["updateComment","creator"]}]`)
		err = c.Post(`mutation{updateComment(comm_id:"1" input:{data:"это изменённый комментарий"}){text id creator{id name about}}}`, &resp, client.AddHeader("user", "2"))
		require.Error(t, err, `[{"message":"can't edit comment of other person","path":["updateComment"]},{"message":"the requested element is null which the schema does not allow","path":["updateComment","creator"]}]`)
		err = c.Post(`mutation{updateComment(comm_id:"1" input:{data:"это изменённый комментарий"}){text id creator{id name about}}}`, &resp, client.AddHeader("user", "21"))
		require.Error(t, err, `[{"message":"user with such id does not exist","path":["updateComment"]},{"message":"the requested element is null which the schema does not allow","path":["updateComment","creator"]}]`)

		c.MustPost(`query{get_comment(comment_id: 1){text creator{id name about}}}`, &resp)
		c.MustPost(`mutation{updateComment(comm_id:"1" input:{data:"это изменённый комментарий"}){text id creator{id name about}}}`, &resp, client.AddHeader("user", "1"))

		require.NotEqual(t, resp.Get_comment.Text, resp.UpdateComment.Text)

		c.MustPost(`query{get_comment(comment_id: 1){text creator{id name about}}}`, &resp)

		require.Equal(t, resp.Get_comment.Text, resp.UpdateComment.Text)
	})

	t.Run("create comment and replies to them and get them", func(t *testing.T) {
		type Resp struct {
			Comments []struct {
				ID      string
				Text    string
				Creator struct {
					ID    string
					Name  string
					About string
				}
				Answer_to string
			}

			Get_replies []struct {
				ID      string
				Text    string
				Creator struct {
					ID    string
					Name  string
					About string
				}
				Answer_to string
			}
		}
		firstPage := Resp{}
		secondPage := Resp{}
		for i := range 30 {
			_, _ = c.RawPost(fmt.Sprintf(`mutation{createComment(input:{text:"это комментарий #%d к посту номер 2" post:2 answer_to: -1}) {id}}`, i+1), client.AddHeader("user", "2"))
		}

		err := c.Post(`query{comments(post_id:2 page: -109) {id}}`, &firstPage)
		require.Error(t, err, `[{"message":"pages start with 1","path":["comments"]}]`)

		c.MustPost(`query{comments(post_id:2 page: 1) {id}}`, &firstPage)
		c.MustPost(`query{comments(post_id:2 page: 2) {id}}`, &secondPage)

		require.Equal(t, len(firstPage.Comments), 20)
		require.Equal(t, len(secondPage.Comments), 10)

		answersFirstPage := Resp{}
		answersSecondPage := Resp{}

		for i := range 30 {
			_, _ = c.RawPost(fmt.Sprintf(`mutation{createComment(input:{text:"это комментарий #%d в ответ на пост 3" post:2 answer_to: 3}) {id}}`, i+1), client.AddHeader("user", "2"))
		}

		err = c.Post(`query{get_replies(comment_id:2 page: -1) {id}}`, &firstPage)
		require.Error(t, err, `[{"message":"pages start with 1","path":["get_replies"]}]`)

		c.MustPost(`query{get_replies(comment_id:3 page: 1) {id}}`, &answersFirstPage)
		c.MustPost(`query{get_replies(comment_id:3 page: 2) {id}}`, &answersSecondPage)

		require.Equal(t, len(answersFirstPage.Get_replies), 20)
		require.Equal(t, len(answersSecondPage.Get_replies), 10)
	})

	t.Run("create uncommentable post and try to comment it", func(t *testing.T) {
		var resp struct {
			CreateComment struct {
				ID string
			}

			CreatePost struct {
				ID string
			}
		}

		c.MustPost(`mutation{createPost(input:{commentable:false data:"это некомментируемый пост"}){id}}`, &resp, client.AddHeader("user", "1"))
		err := c.Post(`mutation{createComment(input:{answer_to: -1 post:31, text: "пытаюсь комментировать закрытый пост"}){id}}`, &resp, client.AddHeader("user", "2"))
		require.Error(t, err, `[{"message":"cannot comment this post (commenting disabled)","path":["createComment"]}]`)
	})
}
