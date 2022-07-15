package helpers

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/seventv/api/internal/gql/v2/gen/model"
	v2structures "github.com/seventv/common/structures/v2"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson"
)

var twitchPictureSizeRegExp = regexp.MustCompile("([0-9]{2,3})x([0-9]{2,3})")

const webpMime = "image/webp"

func EmoteStructureToModel(s structures.Emote, cdnURL string) *model.Emote {
	version, _ := s.GetVersion(s.ID)
	files := []structures.EmoteFile{}

	for _, file := range version.ImageFiles {
		if file.ContentType != webpMime || (version.Animated && file.FrameCount == 1) {
			continue
		}

		files = append(files, file)
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

	width := make([]int, len(files))
	height := make([]int, len(files))
	urls := make([][]string, len(files))

	sort.Slice(files, func(i, j int) bool {
		return files[i].Width < files[j].Width
	})

	for i, file := range files {
		width[i] = int(file.Width)
		height[i] = int(file.Height)
		urls[i] = []string{
			file.Name,
			fmt.Sprintf("https://%s/%s", cdnURL, file.Key),
		}
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
		Owner:        UserStructureToModel(owner, cdnURL),
		Urls:         urls,
		Width:        width,
		Height:       height,
	}
}

func UserStructureToModel(s structures.User, cdnURL string) *model.User {
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
		avatarURL = fmt.Sprintf("//%s/pp/%s/%s", cdnURL, s.ID.Hex(), s.AvatarID)
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
		Role:         RoleStructureToModel(highestRole),
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

func UserStructureToPartialModel(s *model.User) *model.UserPartial {
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

func RoleStructureToModel(s structures.Role) *model.Role {
	p := 0

	bits := map[structures.RolePermission]int64{
		structures.RolePermissionCreateEmote:                    v2structures.RolePermissionEmoteCreate,
		structures.RolePermissionEditEmote:                      v2structures.RolePermissionEmoteEditOwned,
		structures.RolePermissionEditAnyEmote:                   v2structures.RolePermissionEmoteEditAll,
		structures.RolePermissionReportCreate:                   v2structures.RolePermissionCreateReports,
		structures.RolePermissionManageBans:                     v2structures.RolePermissionBanUsers,
		structures.RolePermissionManageUsers:                    v2structures.RolePermissionManageUsers,
		structures.RolePermissionManageStack:                    v2structures.RolePermissionEditApplicationMeta,
		structures.RolePermissionManageCosmetics:                v2structures.RolePermissionManageEntitlements,
		structures.RolePermissionFeatureZeroWidthEmoteType:      v2structures.RolePermissionUseZeroWidthEmote,
		structures.RolePermissionFeatureProfilePictureAnimation: v2structures.RolePermissionUseCustomAvatars,
		structures.RolePermissionSuperAdministrator:             v2structures.RolePermissionAdministrator,
	}

	for a, b := range bits {
		if s.HasPermissionBit(a) {
			p |= int(b)
		}
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

func BanStructureToModel(s *structures.Ban) *model.Ban {
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

func CosmeticStructureToModel(s structures.Cosmetic[bson.Raw]) *model.UserCosmetic {
	switch s.Kind {
	case structures.CosmeticKindNametagPaint:
		v, _ := structures.ConvertCosmetic[structures.CosmeticDataPaint](s)

		f := strings.Replace(string(v.Data.Function), "_", "-", 1)
		f = strings.ToLower(f)
		v.Data.Function = structures.CosmeticPaintFunction(f)

		j, _ := json.Marshal(v.Data)

		return &model.UserCosmetic{
			ID:       v.ID.Hex(),
			Kind:     string(v.Kind),
			Name:     v.Name,
			Selected: v.Selected,
			Data:     utils.B2S(j),
		}
	case structures.CosmeticKindBadge:
		v, _ := structures.ConvertCosmetic[structures.CosmeticDataBadge](s)

		j, _ := json.Marshal(v.Data)

		return &model.UserCosmetic{
			ID:       v.ID.Hex(),
			Kind:     string(v.Kind),
			Name:     v.Name,
			Selected: v.Selected,
			Data:     utils.B2S(j),
		}
	}

	return &model.UserCosmetic{}
}
