package errors

import "fmt"

type ErrorGQL error

var (
	ErrAccessDenied  ErrorGQL = fmt.Errorf("access denied")
	ErrLoginRequired ErrorGQL = fmt.Errorf("login required")
)
