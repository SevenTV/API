package model

import (
	"time"

	"github.com/seventv/api/internal/api/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (xm EmoteModel) GQL() *model.Emote {
	var (
		versions = make([]*model.EmoteVersion, len(xm.Versions))
		owner    *model.UserPartial
	)

	for i, v := range xm.Versions {
		versions[i] = v.GQL()

		if v.ID == xm.ID {
			versions[i].Host = xm.Host.GQL()
		}
	}

	if xm.Owner != nil {
		u := *xm.Owner

		owner = u.GQL()
	}

	return &model.Emote{
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
		Host:      xm.Host.GQL(),
		Versions:  versions,
	}
}

func (xm EmotePartialModel) GQL() *model.EmotePartial {
	var (
		ownerID primitive.ObjectID
		owner   *model.UserPartial
	)

	if xm.Owner != nil {
		u := *xm.Owner

		ownerID = u.ID
		owner = u.GQL()
	}

	return &model.EmotePartial{
		ID:        xm.ID,
		Name:      xm.Name,
		Flags:     int(xm.Flags),
		Listed:    xm.Listed,
		Lifecycle: int(xm.Lifecycle),
		Tags:      xm.Tags,
		Animated:  xm.Animated,
		OwnerID:   ownerID,
		Owner:     owner,
		Host:      xm.Host.GQL(),
	}
}

func (xm EmoteVersionModel) GQL() *model.EmoteVersion {
	host := &model.ImageHost{
		URL:   "",
		Files: []*model.Image{},
	}
	if xm.Host != nil {
		host = xm.Host.GQL()
	}

	return &model.EmoteVersion{
		ID:          xm.ID,
		Name:        xm.Name,
		Description: xm.Description,
		CreatedAt:   time.UnixMilli(xm.CreatedAt),
		Host:        host,
		Lifecycle:   int(xm.Lifecycle),
		Listed:      xm.Listed,
	}
}
