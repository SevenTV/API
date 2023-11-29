package search

import "github.com/meilisearch/meilisearch-go"

type EmoteSearchOptions struct {
	Limit     int64
	Page      int64
	Sort      EmoteSortOptions
	Personal  bool
	Listed    bool
	Lifecycle int32
}

type EmoteSortOptions struct {
	By        string
	Ascending bool
}

type EmoteResult struct {
	Name string
	Id   string
}

func (s *MeiliSearch) SearchEmotes(query string, opt EmoteSearchOptions) ([]EmoteResult, int64, error) {
	req := &meilisearch.SearchRequest{}
	if opt.Limit != 0 {
		req.Limit = opt.Limit
	}
	if opt.Page != 0 {
		req.Page = opt.Page + 1
	}
	if opt.Sort.By != "" {
		req.Sort = []string{opt.Sort.By + ":" + map[bool]string{true: "asc", false: "desc"}[opt.Sort.Ascending]}
	}

	filter := ""

	if opt.Personal {
		filter = "personal = true"
	}
	if opt.Listed {
		if filter != "" {
			filter += " AND "
		}
		filter += "listed = true"
	}
	if opt.Lifecycle != 0 {
		if filter != "" {
			filter += " AND "
		}
		filter += "lifecycle = " + string(opt.Lifecycle)
	}

	if filter != "" {
		req.Filter = filter
	}

	res, err := s.emoteIndex.Search(query, req)

	if err != nil {
		return nil, 0, err
	}

	var hit map[string]interface{}
	var emotes []EmoteResult

	for _, result := range res.Hits {
		hit = result.(map[string]interface{})
		emotes = append(emotes, EmoteResult{
			Name: hit["name"].(string),
			Id:   hit["id"].(string),
		})
	}

	return emotes, res.TotalHits, nil
}
