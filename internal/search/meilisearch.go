package search

import (
	"github.com/meilisearch/meilisearch-go"

	"github.com/seventv/api/internal/configure"
)

type MeiliSearch struct {
	emoteIndex *meilisearch.Index
}

func New(cfg *configure.Config) *MeiliSearch {
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   cfg.Meilisearch.Host,
		APIKey: cfg.Meilisearch.Key,
	})

	index := client.Index(cfg.Meilisearch.Index)

	return &MeiliSearch{index}
}
