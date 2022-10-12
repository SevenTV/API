package model

import (
	"time"

	"github.com/seventv/api/internal/gql/v3/gen/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (xm EmoteModel) GQL() *model.Emote {
	var (
		versions = make([]*model.EmoteVersion, len(xm.Versions))
		ownerID  primitive.ObjectID
		owner    *model.UserPartial
	)

	for i, v := range xm.Versions {
		versions[i] = v.GQL()
	}

	if xm.Owner != nil {
		u := *xm.Owner

		ownerID = u.ID
		owner = u.GQL()
	}

	return &model.Emote{
		ID:        xm.ID,
		Name:      xm.Name,
		Flags:     int(xm.Flags),
		Lifecycle: int(xm.Lifecycle),
		Tags:      xm.Tags,
		Animated:  xm.Animated,
		CreatedAt: xm.ID.Timestamp(),
		OwnerID:   ownerID,
		Owner:     owner,
		Host:      xm.Host.GQL(),
		Versions:  versions,
		Listed:    xm.Listed,
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
		Lifecycle: int(xm.Lifecycle),
		Tags:      xm.Tags,
		Animated:  xm.Animated,
		OwnerID:   ownerID,
		Owner:     owner,
		Host:      xm.Host.GQL(),
	}
}

func (xm EmoteVersionModel) GQL() *model.EmoteVersion {
	return &model.EmoteVersion{
		ID:          xm.ID,
		Name:        xm.Name,
		Description: xm.Description,
		CreatedAt:   time.UnixMilli(xm.CreatedAt),
		Host:        xm.Host.GQL(),
		Lifecycle:   int(xm.Lifecycle),
		Listed:      xm.Listed,
	}
}
