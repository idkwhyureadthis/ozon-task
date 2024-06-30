package auth

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/idkwhyureadthis/ozon-task/internal/pkg/isnumber"
)

func IsAuthorized(ctx context.Context) (string, error) {
	userId := ctx.Value("user")
	if userId == nil {
		graphql.AddErrorf(ctx, "not authorized")
		return " ", errors.New("user not authorized")
	}
	if !isnumber.IsNumber(userId.(string)) {
		graphql.AddErrorf(ctx, "wrong user id")
		return " ", errors.New("wrong user id provided in context")
	}
	return ctx.Value("user").(string), nil
}
