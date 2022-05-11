package helpers

import "fmt"

type ErrorGQL error

var (
	ErrUnauthorized        ErrorGQL = fmt.Errorf("unauthorized")
	ErrAccessDenied        ErrorGQL = fmt.Errorf("access denied")
	ErrUnknownEmote        ErrorGQL = fmt.Errorf("unknown emote")
	ErrUnknownUser         ErrorGQL = fmt.Errorf("unknown user")
	ErrUnknownRole         ErrorGQL = fmt.Errorf("unknown role")
	ErrUnknownReport       ErrorGQL = fmt.Errorf("unknown report")
	ErrBadObjectID         ErrorGQL = fmt.Errorf("bad object id")
	ErrInternalServerError ErrorGQL = fmt.Errorf("internal server error")
	ErrBadInt              ErrorGQL = fmt.Errorf("bad int")
	ErrDontBeSilly         ErrorGQL = fmt.Errorf("don't be silly")
)
