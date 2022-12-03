package models

import (
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func MarshalObjectID(id primitive.ObjectID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = w.Write([]byte(fmt.Sprintf(`"%s"`, id.Hex())))
	})
}

func UnmarshalObjectID(v interface{}) (primitive.ObjectID, error) {
	switch v := v.(type) {
	case string:
		return primitive.ObjectIDFromHex(v)
	default:
		return primitive.NilObjectID, fmt.Errorf("%T is not an ObjectID", v)
	}
}
