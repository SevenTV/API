package search

import (
	"github.com/meilisearch/meilisearch-go"

	"github.com/seventv/api/internal/global"
)

type MeiliSearch struct {
	emoteIndex *meilisearch.Index
}

func New(gctx global.Context) *MeiliSearch {
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   gctx.Config().Meilisearch.Host,
		APIKey: gctx.Config().Meilisearch.Key,
	})

	index := client.Index(gctx.Config().Meilisearch.Index)

	return &MeiliSearch{index}
}
