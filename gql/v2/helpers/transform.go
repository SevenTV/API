package helpers

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	v2structures "github.com/SevenTV/Common/structures/v2"
	"github.com/SevenTV/Common/structures/v3"
	"github.com/SevenTV/Common/utils"
	"github.com/seventv/api/global"
	"github.com/seventv/api/gql/v2/gen/model"
)

var twitchPictureSizeRegExp = regexp.MustCompile("([0-9]{2,3})x([0-9]{2,3})")

func EmoteStructureToModel(ctx global.Context, s structures.Emote) *model.Emote {
	version, _ := s.GetVersion(s.ID)

	width := make([]int, 4)
	height := make([]int, 4)
	urls := make([][]string, 4)
	for _, format := range version.Formats {
		if format.Name != structures.EmoteFormatNameWEBP {
			continue
		}
		pos := 0
		for _, f := range format.Files {
			if version.FrameCount > 1 && !f.Animated || pos > 4 {
				continue
			}

			width[pos] = int(f.Width)
			height[pos] = int(f.Height)
			urls[pos] = []string{
				fmt.Sprintf("%d", pos+1),
				fmt.Sprintf("//%s/emote/%s/%s", ctx.Config().CdnURL, version.ID.Hex(), f.Name),
			}
			pos++
		}
	}

	vis := 0
	if !version.State.Listed {
		vis |= int(v2structures.EmoteVisibilityUnlisted)
	}
	if utils.BitField.HasBits(int64(s.Flags), int64(structures.EmoteFlagsZeroWidth)) {
		vis |= int(v2structures.EmoteVisibilityZeroWidth)
	}
	if utils.BitField.HasBits(int64(s.Flags), int64(structures.EmoteFlagsPrivate)) {
		vis |= int(v2structures.EmoteVisibilityPrivate)
	}

	owner := structures.DeletedUser
	if s.Owner != nil {
		owner = *s.Owner
	}

	return &model.Emote{
		ID:           s.ID.Hex(),
		Name:         s.Name,
		OwnerID:      s.OwnerID.Hex(),
		Visibility:   vis,
		Mime:         "image/webp",
		Status:       int(version.State.Lifecycle),
		Tags:         s.Tags,
		CreatedAt:    s.ID.Timestamp().Format(time.RFC3339),
		AuditEntries: []*model.AuditLog{},
		Channels:     []*model.UserPartial{},
		ChannelCount: int(version.State.ChannelCount),
		Owner:        UserStructureToModel(ctx, owner),
		Urls:         urls,
		Width:        width,
		Height:       width,
	}
}

func UserStructureToModel(ctx global.Context, s structures.User) *model.User {
	highestRole := s.GetHighestRole()
	rank := 0
	if !highestRole.ID.IsZero() {
		rank = int(highestRole.Position)
		highestRole.Allowed = s.FinalPermission()
		highestRole.Denied = 0
	} else {
		highestRole = structures.NilRole
	}

	// The emote set attached to twitch connection
	// his is a non-v2 property added to facilitate fetching of "user emotes"
	emoteSetID := ""

	// Twitch/YT connections
	twConn, _, err := s.Connections.Twitch()
	if err == nil {
		emoteSetID = twConn.EmoteSetID.Hex()
	}

	// Avatar URL
	avatarURL := ""
	if s.AvatarID != "" {
		avatarURL = fmt.Sprintf("//%s/pp/%s/%s", ctx.Config().CdnURL, s.ID.Hex(), s.AvatarID)
	}

	// Editors
	editorIds := make([]string, len(s.Editors))
	for i, ed := range s.Editors {
		// ignore if no permission to manage active emotes
		// (this is the only editor permission in v2)
		if !ed.HasPermission(structures.UserEditorPermissionModifyEmotes) {
			continue
		}
		editorIds[i] = ed.ID.Hex()
	}

	user := &model.User{
		ID:           s.ID.Hex(),
		Email:        nil,
		Description:  s.Biography,
		Rank:         rank,
		Role:         RoleStructureToModel(ctx, highestRole),
		EmoteIds:     []string{},
		EmoteAliases: [][]string{},
		// EditorIds:         []string{},
		CreatedAt:       s.ID.Timestamp().Format(time.RFC3339),
		DisplayName:     s.DisplayName,
		Login:           s.Username,
		ProfileImageURL: avatarURL,
		EmoteSetID:      emoteSetID,
		// Emotes:            []*model.Emote{},
		OwnedEmotes:      []*model.Emote{},
		ThirdPartyEmotes: []*model.Emote{},
		EditorIds:        editorIds,
		// EditorIn:          []*model.UserPartial{},
		// Reports:           []*model.Report{},
		// AuditEntries:      []*model.AuditLog{},
		// Bans:              []*model.Ban{},
		// Banned:            false,
		// FollowerCount:     0,
		// Broadcast:         &model.Broadcast{},
		// Notifications:     []*model.Notification{},
		// NotificationCount: 0,
		// Cosmetics:         []*model.UserCosmetic{},
	}
	user.TwitchID = twConn.ID
	user.EmoteSlots = int(twConn.EmoteSlots)
	user.BroadcasterType = twConn.Data.BroadcasterType

	// set avatar url to twitch cdn if none set in app
	if avatarURL == "" && len(twConn.Data.ProfileImageURL) >= 6 {
		user.ProfileImageURL = twitchPictureSizeRegExp.ReplaceAllString(twConn.Data.ProfileImageURL[6:], "70x70")
	}

	if ytConn, _, err := s.Connections.YouTube(); err == nil {
		user.YoutubeID = ytConn.ID
	}

	return user
}

func UserStructureToPartialModel(ctx global.Context, s *model.User) *model.UserPartial {
	return &model.UserPartial{
		ID:              s.ID,
		Rank:            s.Rank,
		Role:            s.Role,
		EmoteIds:        s.EmoteIds,
		EmoteSetID:      s.EmoteSetID,
		EditorIds:       s.EditorIds,
		CreatedAt:       s.CreatedAt,
		TwitchID:        s.TwitchID,
		DisplayName:     s.DisplayName,
		Login:           s.Login,
		ProfileImageURL: s.ProfileImageURL,
	}
}

func RoleStructureToModel(ctx global.Context, s structures.Role) *model.Role {
	p := 0
	switch s.Allowed {
	case structures.RolePermissionCreateEmote:
		p |= int(v2structures.RolePermissionEmoteCreate)
	case structures.RolePermissionEditEmote:
		p |= int(v2structures.RolePermissionEmoteEditOwned)
	case structures.RolePermissionEditAnyEmote:
		p |= int(v2structures.RolePermissionEmoteEditAll)
	case structures.RolePermissionReportCreate:
		p |= int(v2structures.RolePermissionCreateReports)
	case structures.RolePermissionManageBans:
		p |= int(v2structures.RolePermissionBanUsers)
	case structures.RolePermissionSuperAdministrator:
		p |= int(v2structures.RolePermissionAdministrator)
	case structures.RolePermissionManageRoles:
		p |= int(v2structures.RolePermissionManageRoles)
	case structures.RolePermissionManageUsers:
		p |= int(v2structures.RolePermissionManageUsers)
	case structures.RolePermissionManageStack:
		p |= int(v2structures.RolePermissionEditApplicationMeta)
	case structures.RolePermissionManageCosmetics:
		p |= int(v2structures.RolePermissionManageEntitlements)
	case structures.RolePermissionFeatureZeroWidthEmoteType:
		p |= int(v2structures.EmoteVisibilityZeroWidth)
	case structures.RolePermissionFeatureProfilePictureAnimation:
		p |= int(v2structures.RolePermissionUseCustomAvatars)
	}

	return &model.Role{
		ID:       s.ID.Hex(),
		Name:     s.Name,
		Position: int(s.Position),
		Color:    int(s.Color),
		Allowed:  strconv.Itoa(p),
		Denied:   "0",
	}
}

func BanStructureToModel(ctx global.Context, s *structures.Ban) *model.Ban {
	victimID := s.VictimID.Hex()
	actorID := s.ActorID.Hex()
	return &model.Ban{
		ID:         s.ID.Hex(),
		UserID:     &victimID,
		Reason:     s.Reason,
		Active:     s.ExpireAt.After(time.Now()),
		IssuedByID: &actorID,
	}
}
