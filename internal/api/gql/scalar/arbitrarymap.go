package models

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
)

func MarshalArbitraryMap(m map[string]any) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		j, _ := json.Marshal(m)
		_, _ = w.Write(j)
	})
}

func UnmarshalArbitraryMap(m any) (map[string]any, error) {
	switch v := m.(type) {
	case map[string]any:
		return v, nil
	default:
		return nil, fmt.Errorf("%T is not map[string]any", v)
	}
}
