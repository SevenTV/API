package modelgql

import (
	"time"

	"github.com/seventv/api/data/model"
	gql_model "github.com/seventv/api/internal/api/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func EmoteModel(xm model.EmoteModel) *gql_model.Emote {
	var (
		versions = make([]*gql_model.EmoteVersion, len(xm.Versions))
		owner    *gql_model.UserPartial
	)

	for i, v := range xm.Versions {
		versions[i] = EmoteVersionModel(v)

		if v.ID == xm.ID {
			versions[i].Host = ImageHost(xm.Host)
		}
	}

	if xm.Owner != nil {
		u := *xm.Owner

		owner = UserPartialModel(u)
	}

	return &gql_model.Emote{
		ID:        xm.ID,
		Name:      xm.Name,
		Flags:     int(xm.Flags),
		Listed:    xm.Listed,
		Lifecycle: int(xm.Lifecycle),
		Tags:      xm.Tags,
		Animated:  xm.Animated,
		CreatedAt: xm.ID.Timestamp(),
		OwnerID:   xm.OwnerID,
		Owner:     owner,
		Host:      ImageHost(xm.Host),
		Versions:  versions,
	}
}

func EmotePartialModel(xm model.EmotePartialModel) *gql_model.EmotePartial {
	var (
		ownerID primitive.ObjectID
		owner   *gql_model.UserPartial
	)

	if xm.Owner != nil {
		u := *xm.Owner

		ownerID = u.ID
		owner = UserPartialModel(u)
	}

	return &gql_model.EmotePartial{
		ID:        xm.ID,
		Name:      xm.Name,
		Flags:     int(xm.Flags),
		Listed:    xm.Listed,
		Lifecycle: int(xm.Lifecycle),
		Tags:      xm.Tags,
		Animated:  xm.Animated,
		OwnerID:   ownerID,
		Owner:     owner,
		Host:      ImageHost(xm.Host),
	}
}

func EmoteVersionModel(xm model.EmoteVersionModel) *gql_model.EmoteVersion {
	host := &gql_model.ImageHost{
		URL:   "",
		Files: []*gql_model.Image{},
	}
	if xm.Host != nil {
		host = ImageHost(*xm.Host)
	}

	return &gql_model.EmoteVersion{
		ID:          xm.ID,
		Name:        xm.Name,
		Description: xm.Description,
		CreatedAt:   time.UnixMilli(xm.CreatedAt),
		Host:        host,
		Lifecycle:   int(xm.Lifecycle),
		Listed:      xm.Listed,
	}
}
