package auth

import (
	"context"

	"github.com/seventv/api/internal/gql/v3/helpers"
	"github.com/seventv/common/structures/v3"
)

func For(ctx context.Context) *structures.User {
	raw, _ := ctx.Value(helpers.UserKey).(*structures.User)
	return raw
}
