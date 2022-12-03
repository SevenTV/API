package helpers

type ErrorGQL string

func (e ErrorGQL) Error() string {
	return string(e)
}

const (
	ErrUnauthorized        ErrorGQL = "unauthorized"
	ErrAccessDenied        ErrorGQL = "access denied"
	ErrUnknownEmote        ErrorGQL = "unknown emote"
	ErrUnknownUser         ErrorGQL = "unknown user"
	ErrUnknownRole         ErrorGQL = "unknown role"
	ErrUnknownReport       ErrorGQL = "unknown report"
	ErrBadObjectID         ErrorGQL = "bad object id"
	ErrInternalServerError ErrorGQL = "internal server error"
	ErrBadInt              ErrorGQL = "bad int"
	ErrDontBeSilly         ErrorGQL = "don't be silly"
)
