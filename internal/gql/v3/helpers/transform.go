package helpers

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/seventv/api/internal/gql/v3/gen/model"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

var twitchPictureSizeRegExp = regexp.MustCompile("([0-9]{2,3})x([0-9]{2,3})")

// UserStructureToModel: Transform a user structure to a GQL mdoel
func UserStructureToModel(s structures.User, cdnURL string) *model.User {
	tagColor := 0
	if role := s.GetHighestRole(); !role.ID.IsZero() {
		tagColor = int(role.Color)
	}
	roles := make([]*model.Role, len(s.Roles))
	for i, v := range s.Roles {
		roles[i] = RoleStructureToModel(v)
	}

	connections := make([]*model.UserConnection, len(s.Connections))
	for i, v := range s.Connections {
		connections[i] = UserConnectionStructureToModel(v)
	}

	editors := make([]*model.UserEditor, len(s.Editors))
	for i, v := range s.Editors {
		editors[i] = UserEditorStructureToModel(v, cdnURL)
	}

	avatarURL := ""
	if s.AvatarID != "" {
		avatarURL = fmt.Sprintf("//%s/pp/%s/%s", cdnURL, s.ID.Hex(), s.AvatarID)
	} else {
		for _, con := range s.Connections {
			switch con.Platform {
			case structures.UserConnectionPlatformTwitch:
				if con, err := structures.ConvertUserConnection[structures.UserConnectionDataTwitch](con); err == nil {
					avatarURL = twitchPictureSizeRegExp.ReplaceAllString(con.Data.ProfileImageURL[6:], "70x70")
				}
			}
		}
	}

	return &model.User{
		ID:               s.ID,
		UserType:         string(s.UserType),
		Username:         s.Username,
		DisplayName:      utils.Ternary(len(s.DisplayName) > 0, s.DisplayName, s.Username),
		CreatedAt:        s.ID.Timestamp(),
		AvatarURL:        avatarURL,
		Biography:        s.Biography,
		TagColor:         tagColor,
		Editors:          editors,
		Roles:            roles,
		OwnedEmotes:      []*model.Emote{},
		Connections:      connections,
		InboxUnreadCount: 0,
		Reports:          []*model.Report{},
	}
}

func UserStructureToPartialModel(m *model.User) *model.UserPartial {
	return &model.UserPartial{
		ID:          m.ID,
		UserType:    m.UserType,
		Username:    m.Username,
		DisplayName: m.DisplayName,
		CreatedAt:   m.ID.Timestamp(),
		AvatarURL:   m.AvatarURL,
		Biography:   m.Biography,
		TagColor:    m.TagColor,
		Roles:       m.Roles,
		Connections: m.Connections,
	}
}

// UserEditorStructureToModel: Transform a user editor structure to a GQL model
func UserEditorStructureToModel(s structures.UserEditor, cdnURL string) *model.UserEditor {
	if s.User == nil {
		s.User = &structures.DeletedUser
	}

	return &model.UserEditor{
		ID:          s.ID,
		Permissions: int(s.Permissions),
		Visible:     s.Visible,
		AddedAt:     s.AddedAt,
		User:        UserStructureToPartialModel(UserStructureToModel(*s.User, cdnURL)),
	}
}

// UserConnectionStructureToModel: Transform a user connection structure to a GQL model
func UserConnectionStructureToModel(s structures.UserConnection[bson.Raw]) *model.UserConnection {
	var (
		err         error
		displayName string
	)
	// Decode the connection data
	switch s.Platform {
	case structures.UserConnectionPlatformTwitch:
		if s, err := structures.ConvertUserConnection[structures.UserConnectionDataTwitch](s); err == nil {
			displayName = s.Data.DisplayName
		}
	case structures.UserConnectionPlatformYouTube:
		if s, err := structures.ConvertUserConnection[structures.UserConnectionDataYoutube](s); err == nil {
			displayName = s.Data.Title
		}
	}
	if err != nil {
		zap.S().Errorw("couldn't decode user connection",
			"error", err,
			"platform", s.Platform,
		)
		return nil
	}

	return &model.UserConnection{
		ID:          s.ID,
		DisplayName: displayName,
		Platform:    model.ConnectionPlatform(s.Platform),
		LinkedAt:    s.LinkedAt,
		EmoteSlots:  int(s.EmoteSlots),
		EmoteSetID:  &s.EmoteSetID,
	}
}

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

	// Sort by version timestamp
	sort.Slice(s.Versions, func(i, j int) bool {
		return s.Versions[i].CreatedAt.After(s.Versions[j].CreatedAt)
	})
	for _, ver := range s.Versions {
		if ver.State.Lifecycle < structures.EmoteLifecycleProcessing || ver.IsUnavailable() {
			continue // skip if lifecycle isn't past pending
		}

		files := ver.GetFiles("", true)
		vimages := make([]*model.Image, len(files))
		for i, fi := range files {
			url := fmt.Sprintf("//%s/emote/%s/%s", cdnURL, ver.ID.Hex(), fi.Name)
			img := EmoteFileStructureToModel(&fi, url)
			vimages[i] = img
		}

		if ver.ID == s.ID {
			lifecycle = ver.State.Lifecycle
			listed = ver.State.Listed
			images = vimages
		}
		versions[versionCount] = EmoteVersionStructureToModel(ver, vimages)
		versionCount++
	}
	if len(versions) != int(versionCount) {
		versions = versions[0:versionCount]
	}

	owner := structures.DeletedUser
	if s.Owner != nil {
		owner = *s.Owner
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
		Owner:     UserStructureToModel(owner, cdnURL),
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
	for i, e := range s.Emotes {
		if e.Emote == nil {
			e.Emote = &structures.DeletedEmote
		}
		emotes[i] = &model.ActiveEmote{
			ID:        e.ID,
			Name:      e.Name,
			Flags:     int(e.Flags),
			Timestamp: e.Timestamp,
			Emote:     EmoteStructureToModel(*e.Emote, cdnURL),
		}
	}
	var owner *model.User
	if s.Owner != nil {
		owner = UserStructureToModel(*s.Owner, cdnURL)
	}

	return &model.EmoteSet{
		ID:         s.ID,
		Name:       s.Name,
		Tags:       s.Tags,
		Emotes:     emotes,
		EmoteSlots: int(s.EmoteSlots),
		OwnerID:    &s.OwnerID,
		Owner:      owner,
	}
}

func EmoteVersionStructureToModel(s structures.EmoteVersion, images []*model.Image) *model.EmoteVersion {
	return &model.EmoteVersion{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Timestamp:   s.ID.Timestamp(),
		Images:      images,
		Lifecycle:   int(s.State.Lifecycle),
		Listed:      s.State.Listed,
	}
}

func EmoteFileStructureToModel(s *structures.EmoteFile, url string) *model.Image {
	return &model.Image{
		Name:        s.Name,
		URL:         url,
		Width:       int(s.Width),
		Height:      int(s.Height),
		ContentType: s.ContentType,
		FrameCount:  int(s.FrameCount),
		Size:        int(s.Size),
		Sha3:        s.SHA3,
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
	author := structures.DeletedUser
	if s.Author != nil {
		author = *s.Author
	}
	return &model.InboxMessage{
		ID:           s.ID,
		Kind:         model.MessageKind(s.Kind.String()),
		CreatedAt:    s.CreatedAt,
		Author:       UserStructureToModel(author, cdnURL),
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
	author := structures.DeletedUser
	if s.Author != nil {
		author = *s.Author
	}
	return &model.ModRequestMessage{
		ID:         s.ID,
		Kind:       model.MessageKind(s.Kind.String()),
		CreatedAt:  s.CreatedAt,
		Author:     UserStructureToModel(author, cdnURL),
		TargetKind: int(s.Data.TargetKind),
		TargetID:   s.Data.TargetID,
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
