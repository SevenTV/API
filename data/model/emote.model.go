package model

import (
	"sort"

	"github.com/seventv/common/structures/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmoteModel struct {
	ID        primitive.ObjectID  `json:"id"`
	Name      string              `json:"name"`
	Flags     EmoteFlagsModel     `json:"flags"`
	Tags      []string            `json:"tags"`
	Lifecycle EmoteLifecycleModel `json:"lifecycle"`
	Listed    bool                `json:"listed"`
	Animated  bool                `json:"animated"`
	Owner     *UserModel          `json:"owner,omitempty" extensions:"x-omitempty"`
	Images    []Image             `json:"images"`
	Versions  []EmoteVersionModel `json:"versions"`
}

type EmoteVersionModel struct {
	ID          primitive.ObjectID  `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Lifecycle   EmoteLifecycleModel `json:"lifecycle"`
	Listed      bool                `json:"listed"`
	Animated    bool                `json:"animated"`
	Images      []Image             `json:"images"`
}

type EmoteLifecycleModel int32

const (
	EmoteLifecycleDeleted EmoteLifecycleModel = iota - 1
	EmoteLifecyclePending
	EmoteLifecycleProcessing
	EmoteLifecycleDisabled
	EmoteLifecycleLive
	EmoteLifecycleFailed EmoteLifecycleModel = -2
)

type EmoteFlagsModel int32

const (
	EmoteFlagsPrivate   EmoteFlagsModel = 1 << 0 // The emote is private and can only be accessed by its owner, editors and moderators
	EmoteFlagsAuthentic EmoteFlagsModel = 1 << 1 // The emote was verified to be an original creation by the uploader
	EmoteFlagsZeroWidth EmoteFlagsModel = 1 << 8 // The emote is recommended to be enabled as Zero-Width

	// Content Flags

	EmoteFlagsContentSexual           EmoteFlagsModel = 1 << 16 // Sexually Suggesive
	EmoteFlagsContentEpilepsy         EmoteFlagsModel = 1 << 17 // Rapid flashing
	EmoteFlagsContentEdgy             EmoteFlagsModel = 1 << 18 // Edgy or distasteful, may be offensive to some users
	EmoteFlagsContentTwitchDisallowed EmoteFlagsModel = 1 << 24 // Not allowed specifically on the Twitch platform
)

func (x *modelizer) Emote(v structures.Emote) EmoteModel {
	images := make([]Image, 0)
	lifecycle := EmoteLifecycleDisabled
	listed := false
	animated := false

	versions := make([]EmoteVersionModel, len(v.Versions))

	for i, ver := range v.Versions {
		files := ver.GetFiles("", true)
		sort.Slice(files, func(i, j int) bool {
			return files[i].Width > files[j].Width
		})

		vimages := make([]Image, len(files))

		for i, fi := range files {
			vimages[i] = x.Image(fi)
		}

		if ver.ID == v.ID {
			lifecycle = EmoteLifecycleModel(ver.State.Lifecycle)
			listed = ver.State.Listed
			animated = ver.Animated
			images = vimages
		}

		versions[i] = x.EmoteVersion(ver)
	}

	var owner *UserModel

	if v.Owner != nil {
		u := x.User(*v.Owner)
		owner = &u
	}

	return EmoteModel{
		ID:        v.ID,
		Name:      v.Name,
		Flags:     EmoteFlagsModel(v.Flags),
		Tags:      v.Tags,
		Lifecycle: lifecycle,
		Listed:    listed,
		Animated:  animated,
		Owner:     owner,
		Images:    images,
		Versions:  versions,
	}
}

func (x *modelizer) EmoteVersion(v structures.EmoteVersion) EmoteVersionModel {
	var images []Image

	for _, fi := range v.GetFiles("", true) {
		images = append(images, x.Image(fi))
	}

	return EmoteVersionModel{
		ID:          v.ID,
		Name:        v.Name,
		Description: v.Description,
		Lifecycle:   EmoteLifecycleModel(v.State.Lifecycle),
		Listed:      v.State.Listed,
		Animated:    v.Animated,
		Images:      images,
	}
}
