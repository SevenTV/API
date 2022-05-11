package auth

import (
	"context"

	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/gql/v3/helpers"
)

func For(ctx context.Context) *structures.User {
	raw, _ := ctx.Value(helpers.UserKey).(*structures.User)
	return raw
}
