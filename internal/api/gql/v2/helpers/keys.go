package helpers

import "github.com/seventv/common/utils"

const (
	UserKey       = utils.Key("user")
	RequestCtxKey = utils.Key("requestCtx")
	StatusGQL2    = utils.Key("gqlStatus") // used to make the status code 400 for gql v2
)
