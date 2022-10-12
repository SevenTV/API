package helpers

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
)

// RoleStructureToModel: Transform a role structure to a GQL model
func RoleStructureToModel(s structures.Role) *model.Role {
	return &model.Role{
		ID:        s.ID,
		Name:      s.Name,
		Color:     int(s.Color),
		Allowed:   strconv.Itoa(int(s.Allowed)),
		Denied:    strconv.Itoa(int(s.Denied)),
		Position:  int(s.Position),
		CreatedAt: s.ID.Timestamp(),
		Invisible: s.Invisible,
		Members:   []*model.User{},
	}
}

func EmoteStructureToModel(s structures.Emote, cdnURL string) *model.Emote {
	images := make([]*model.Image, 0)
	versions := make([]*model.EmoteVersion, len(s.Versions))
	versionCount := int32(0)
	lifecycle := structures.EmoteLifecycleDisabled
	listed := false
	animated := false

	for _, ver := range s.Versions {
		files := ver.GetFiles("", true)
		sort.Slice(files, func(i, j int) bool {
			return files[i].Width < files[j].Width
		})

		vimages := make([]*model.Image, len(files))

		for i, fi := range files {
			url := fmt.Sprintf("//%s/%s", cdnURL, fi.Key)
			img := EmoteFileStructureToModel(&fi, url)
			vimages[i] = img
		}

		if ver.ID == s.ID {
			lifecycle = ver.State.Lifecycle
			listed = ver.State.Listed
			images = vimages
			animated = ver.Animated
		}

		if !ver.IsUnavailable() {
			archive := EmoteFileStructureToArchiveModel(ver.ArchiveFile, fmt.Sprintf("//%s/%s", cdnURL, ver.ArchiveFile.Key))
			versions[versionCount] = EmoteVersionStructureToModel(ver, vimages, archive)
			versionCount++
		}
	}

	if len(versions) != int(versionCount) {
		versions = versions[0:versionCount]
	}

	return &model.Emote{
		ID:        s.ID,
		Name:      s.Name,
		Flags:     int(s.Flags),
		Lifecycle: int(lifecycle),
		Tags:      s.Tags,
		Animated:  animated,
		CreatedAt: s.ID.Timestamp(),
		OwnerID:   s.OwnerID,
		Channels:  &model.UserSearchResult{},
		Images:    images,
		Versions:  versions,
		Listed:    listed,
		Reports:   []*model.Report{},
	}
}

func EmoteStructureToPartialModel(m *model.Emote) *model.EmotePartial {
	return &model.EmotePartial{
		ID:        m.ID,
		Name:      m.Name,
		Flags:     m.Flags,
		Lifecycle: m.Lifecycle,
		Tags:      m.Tags,
		Animated:  m.Animated,
		CreatedAt: m.CreatedAt,
		OwnerID:   m.OwnerID,
		Owner:     m.Owner,
		Images:    m.Images,
		Versions:  m.Versions,
		Listed:    m.Listed,
	}
}

func EmoteSetStructureToModel(s structures.EmoteSet, cdnURL string) *model.EmoteSet {
	emotes := make([]*model.ActiveEmote, len(s.Emotes))

	for i, ae := range s.Emotes {
		if ae.Emote == nil {
			ae.Emote = &structures.DeletedEmote
		}

		emotes[i] = &model.ActiveEmote{
			ID:        ae.ID,
			Name:      ae.Name,
			Flags:     int(ae.Flags),
			Timestamp: ae.Timestamp,
			Emote:     EmoteStructureToModel(*ae.Emote, cdnURL),
			Actor:     &model.UserPartial{ID: ae.ActorID},
		}
	}

	return &model.EmoteSet{
		ID:       s.ID,
		Name:     s.Name,
		Tags:     s.Tags,
		Emotes:   emotes,
		Capacity: int(s.Capacity),
		OwnerID:  &s.OwnerID,
	}
}

func EmoteVersionStructureToModel(s structures.EmoteVersion, images []*model.Image, archive *model.Archive) *model.EmoteVersion {
	return &model.EmoteVersion{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		CreatedAt:   s.CreatedAt,
		StartedAt:   s.StartedAt,
		CompletedAt: s.CompletedAt,
		Images:      images,
		Lifecycle:   int(s.State.Lifecycle),
		Listed:      s.State.Listed,
		Error:       utils.Ternary(s.State.Error == "", nil, &s.State.Error),
		Archive:     archive,
	}
}

func EmoteFileStructureToArchiveModel(s structures.ImageFile, url string) *model.Archive {
	return &model.Archive{
		Name:        s.Name,
		URL:         url,
		ContentType: s.ContentType,
		Size:        int(s.Size),
	}
}

func EmoteFileStructureToModel(s *structures.ImageFile, url string) *model.Image {
	// Transform image format
	var format model.ImageFormat

	switch s.ContentType {
	case "image/avif":
		format = model.ImageFormatAvif
	case "image/webp":
		format = model.ImageFormatWebp
	case "image/gif":
		format = model.ImageFormatGif
	case "image/png":
		format = model.ImageFormatPng
	}

	return &model.Image{
		Name:       s.Name,
		URL:        url,
		Width:      int(s.Width),
		Height:     int(s.Height),
		Format:     format,
		FrameCount: int(s.FrameCount),
		Size:       int(s.Size),
	}
}

func ActiveEmoteStructureToModel(s *structures.ActiveEmote) *model.ActiveEmote {
	return &model.ActiveEmote{
		ID:        s.ID,
		Name:      s.Name,
		Flags:     int(s.Flags),
		Timestamp: s.Timestamp,
	}
}

func MessageStructureToInboxModel(s structures.Message[structures.MessageDataInbox], cdnURL string) *model.InboxMessage {
	return &model.InboxMessage{
		ID:           s.ID,
		Kind:         model.MessageKind(s.Kind.String()),
		CreatedAt:    s.CreatedAt,
		Read:         s.Read,
		ReadAt:       &time.Time{},
		Subject:      s.Data.Subject,
		Content:      s.Data.Content,
		Important:    s.Data.Important,
		Starred:      s.Data.Starred,
		Pinned:       s.Data.Pinned,
		Placeholders: utils.Ternary(s.Data.Placeholders == nil, map[string]string{}, s.Data.Placeholders),
	}
}

func MessageStructureToModRequestModel(s structures.Message[structures.MessageDataModRequest], cdnURL string) *model.ModRequestMessage {
	return &model.ModRequestMessage{
		ID:         s.ID,
		Kind:       model.MessageKind(s.Kind.String()),
		CreatedAt:  s.CreatedAt,
		TargetKind: int(s.Data.TargetKind),
		TargetID:   s.Data.TargetID,
	}
}

func ReportStructureToModel(s structures.Report) *model.Report {
	assignees := make([]*model.User, len(s.AssigneeIDs))
	for i, oid := range s.AssigneeIDs {
		assignees[i] = &model.User{ID: oid}
	}

	return &model.Report{
		ID:         s.ID,
		TargetKind: int(s.TargetKind),
		TargetID:   s.TargetID,
		ActorID:    s.ActorID,
		Subject:    s.Subject,
		Body:       s.Body,
		Priority:   int(s.Priority),
		Status:     model.ReportStatus(s.Status),
		CreatedAt:  s.CreatedAt,
		Notes:      []string{},
		Assignees:  assignees,
	}
}

func BanStructureToModel(s structures.Ban) *model.Ban {
	return &model.Ban{
		ID:        s.ID,
		Reason:    s.Reason,
		Effects:   int(s.Effects),
		ExpireAt:  s.ExpireAt,
		CreatedAt: s.ID.Timestamp(),
		ActorID:   s.ActorID,
		VictimID:  s.VictimID,
	}
}

func CosmeticPaintStructureToModel(s structures.Cosmetic[structures.CosmeticDataPaint]) *model.CosmeticPaint {
	var color *int
	if s.Data.Color != nil {
		color = utils.PointerOf(int(*s.Data.Color))
	}

	stops := make([]*model.CosmeticPaintStop, len(s.Data.Stops))
	for i, sto := range s.Data.Stops {
		stops[i] = &model.CosmeticPaintStop{
			At:    sto.At,
			Color: int(sto.Color),
		}
	}

	shadows := make([]*model.CosmeticPaintShadow, len(s.Data.DropShadows))
	for i, sha := range s.Data.DropShadows {
		shadows[i] = &model.CosmeticPaintShadow{
			XOffset: sha.OffsetX,
			YOffset: sha.OffsetY,
			Radius:  sha.Radius,
			Color:   int(sha.Color),
		}
	}

	return &model.CosmeticPaint{
		ID:       s.ID,
		Kind:     model.CosmeticKind(s.Kind),
		Name:     s.Name,
		Function: model.CosmeticPaintFunction(s.Data.Function),
		Color:    color,
		Angle:    int(s.Data.Angle),
		Shape:    &s.Data.Shape,
		ImageURL: &s.Data.ImageURL,
		Repeat:   s.Data.Repeat,
		Stops:    stops,
		Shadows:  shadows,
	}
}

func CosmeticBadgeStructureToModel(s structures.Cosmetic[structures.CosmeticDataBadge]) *model.CosmeticBadge {
	return &model.CosmeticBadge{
		ID:      s.ID,
		Kind:    model.CosmeticKind(s.Kind),
		Name:    s.Name,
		Tooltip: s.Data.Tooltip,
		Tag:     s.Data.Tag,
		Images:  []*model.Image{},
	}
}
