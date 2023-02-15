package auth

import (
	"context"

	"github.com/seventv/api/internal/constant"
	"github.com/seventv/common/structures/v3"
)

func For(ctx context.Context) structures.User {
	raw, _ := ctx.Value(constant.UserKey).(structures.User)
	return raw
}
