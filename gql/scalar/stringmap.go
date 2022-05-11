package models

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
)

func MarshalStringMap(m map[string]string) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		j, _ := json.Marshal(m)
		_, _ = w.Write(j)
	})
}

func UnmarshalStringMap(m interface{}) (map[string]string, error) {
	switch v := m.(type) {
	case map[string]string:
		return v, nil
	default:
		return nil, fmt.Errorf("%T is not map[string]string", v)
	}
}
