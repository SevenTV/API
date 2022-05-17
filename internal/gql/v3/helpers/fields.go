package helpers

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

type Field struct {
	Name     string
	Children map[string]Field
}

func GetFields(ctx context.Context) map[string]Field {
	return GetNestedPreloads(
		graphql.GetOperationContext(ctx),
		graphql.CollectFieldsCtx(ctx, nil),
	)
}

func GetNestedPreloads(ctx *graphql.OperationContext, fields []graphql.CollectedField) map[string]Field {
	f := map[string]Field{}
	for _, column := range fields {
		f[column.Name] = Field{
			Name:     column.Name,
			Children: GetNestedPreloads(ctx, graphql.CollectFields(ctx, column.Selections, nil)),
		}
	}
	return f
}
