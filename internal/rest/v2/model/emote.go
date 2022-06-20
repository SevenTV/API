package model

import (
	"fmt"
	"sort"

	v2structures "github.com/seventv/common/structures/v2"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
)

type Emote struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	Owner            *User       `json:"owner"`
	Visibility       int32       `json:"visibility"`
	VisibilitySimple []string    `json:"visibility_simple"`
	Mime             string      `json:"mime"`
	Status           int8        `json:"status"`
	Tags             []string    `json:"tags"`
	Width            []int32     `json:"width"`
	Height           []int32     `json:"height"`
	URLs             [][2]string `json:"urls"`
}

const webpMime = "image/webp"

func NewEmote(s structures.Emote, cdnURL string) *Emote {
	version, _ := s.GetVersion(s.ID)
	files := []structures.EmoteFile{}
	status := structures.EmoteLifecycle(0)
	if !version.ID.IsZero() {
		for _, file := range version.ImageFiles {
			if file.ContentType != webpMime || (version.Animated && file.FrameCount == 1) {
				continue
			}

			files = append(files, file)
		}
		status = version.State.Lifecycle
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

	simpleVis := []string{}
	for v, s := range v2structures.EmoteVisibilitySimpleMap {
		if !utils.BitField.HasBits(int64(vis), int64(v)) {
			continue
		}

		simpleVis = append(simpleVis, s)
	}

	owner := structures.DeletedUser
	if s.Owner != nil {
		owner = *s.Owner
	}

	width := make([]int32, len(files))
	height := make([]int32, len(files))
	urls := make([][2]string, len(files))

	sort.Slice(files, func(i, j int) bool {
		return files[i].Width > files[j].Width
	})

	for i, file := range files {
		width[i] = file.Width
		height[i] = file.Height
		urls[i] = [2]string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("https://%s/emote/%s/%s", cdnURL, version.ID.Hex(), file.Name),
		}
	}

	return &Emote{
		ID:               s.ID.Hex(),
		Name:             s.Name,
		Owner:            NewUser(owner),
		Visibility:       int32(vis),
		VisibilitySimple: simpleVis,
		Mime:             webpMime,
		Status:           int8(status),
		Tags:             s.Tags,
		Width:            width,
		Height:           height,
		URLs:             urls,
	}
}
