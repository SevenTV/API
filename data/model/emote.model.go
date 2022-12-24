package model

import (
	"fmt"
	"sort"

	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmoteModel struct {
	ID        primitive.ObjectID  `json:"id"`
	Name      string              `json:"name"`
	Flags     EmoteFlagsModel     `json:"flags"`
	Tags      []string            `json:"tags"`
	Lifecycle EmoteLifecycleModel `json:"lifecycle"`
	States    []EmoteVersionState `json:"states"`
	Listed    bool                `json:"listed"`
	Animated  bool                `json:"animated"`
	Owner     *UserPartialModel   `json:"owner,omitempty" extensions:"x-omitempty"`
	OwnerID   primitive.ObjectID  `json:"-"`
	Host      ImageHost           `json:"host"`
	Versions  []EmoteVersionModel `json:"versions"`
}

type EmotePartialModel struct {
	ID        primitive.ObjectID  `json:"id"`
	Name      string              `json:"name"`
	Flags     EmoteFlagsModel     `json:"flags"`
	Tags      []string            `json:"tags,omitempty"`
	Lifecycle EmoteLifecycleModel `json:"lifecycle"`
	States    []EmoteVersionState `json:"states"`
	Listed    bool                `json:"listed"`
	Animated  bool                `json:"animated"`
	Owner     *UserPartialModel   `json:"owner,omitempty" extensions:"x-omitempty"`
	Host      ImageHost           `json:"host"`
}

type EmoteVersionModel struct {
	ID          primitive.ObjectID  `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Lifecycle   EmoteLifecycleModel `json:"lifecycle"`
	States      []EmoteVersionState `json:"states"`
	Listed      bool                `json:"listed"`
	Animated    bool                `json:"animated"`
	Host        *ImageHost          `json:"host,omitempty" extensions:"x-omitempty"`
	CreatedAt   int64               `json:"createdAt"`
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

type EmoteVersionState string

const (
	EmoteVersionStateListed        EmoteVersionState = "LISTED"
	EmoteVersionStateAllowPersonal EmoteVersionState = "ALLOW_PERSONAL"
)

func (x *modelizer) Emote(v structures.Emote) EmoteModel {
	images := make([]ImageFile, 0)
	lifecycle := EmoteLifecycleDisabled
	listed := false
	animated := false

	states := make(utils.Set[EmoteVersionState], 0)

	versions := make([]EmoteVersionModel, len(v.Versions))

	for i, ver := range v.Versions {
		vimages := make([]ImageFile, 0)

		if ver.State.Lifecycle == structures.EmoteLifecycleLive {
			files := append(ver.GetFiles("image/avif", true), ver.GetFiles("image/webp", true)...)
			sort.Slice(files, func(i, j int) bool {
				return files[i].Width < files[j].Width
			})

			vimages = make([]ImageFile, len(files))

			for i, fi := range files {
				vimages[i] = x.Image(fi)
			}
		}

		if ver.ID == v.ID {
			lifecycle = EmoteLifecycleModel(ver.State.Lifecycle)
			animated = ver.Animated
			images = vimages

			if listed = ver.State.Listed; listed {
				states.Add(EmoteVersionStateListed)
			}

			if ver.State.AllowPersonal != nil && *ver.State.AllowPersonal {
				states.Add(EmoteVersionStateAllowPersonal)
			}
		}

		if ver.IsUnavailable() {
			continue
		}

		versions[i] = x.EmoteVersion(ver)
	}

	var owner *UserPartialModel

	if v.Owner != nil {
		u := x.User(*v.Owner).ToPartial()
		u.Connections = nil // clear the connections field of emote owners as it's not needed here

		owner = &u
	}

	if len(versions) > 0 {
		// Sort versions
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].ID == v.ID || versions[j].ID.Timestamp().After(versions[i].ID.Timestamp())
		})

		// Remove the image host from versions[0]
		// (it would be redundant with the top-level property)
		versions[0].Host = nil
	}

	return EmoteModel{
		ID:        v.ID,
		Name:      v.Name,
		Flags:     EmoteFlagsModel(v.Flags),
		Tags:      v.Tags,
		Lifecycle: lifecycle,
		States:    states.Values(),
		Listed:    listed,
		Animated:  animated,
		Owner:     owner,
		OwnerID:   v.OwnerID,
		Host: ImageHost{
			URL:   fmt.Sprintf("//%s/emote/%s", x.cdnURL, v.ID.Hex()),
			Files: images,
		},
		Versions: versions,
	}
}

func (em EmoteModel) ToPartial() EmotePartialModel {
	return EmotePartialModel{
		ID:        em.ID,
		Name:      em.Name,
		Flags:     em.Flags,
		Tags:      em.Tags,
		Lifecycle: em.Lifecycle,
		States:    em.States,
		Listed:    em.Listed,
		Animated:  em.Animated,
		Owner:     em.Owner,
		Host:      em.Host,
	}
}

func (x *modelizer) EmoteVersion(v structures.EmoteVersion) EmoteVersionModel {
	var files []ImageFile

	for _, fi := range append(v.GetFiles("image/avif", true), v.GetFiles("image/webp", true)...) {
		files = append(files, x.Image(fi))
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Width < files[j].Width
	})

	states := make(utils.Set[EmoteVersionState])

	if v.State.Listed {
		states.Add(EmoteVersionStateListed)
	}

	if v.State.AllowPersonal != nil && *v.State.AllowPersonal {
		states.Add(EmoteVersionStateAllowPersonal)
	}

	return EmoteVersionModel{
		ID:          v.ID,
		Name:        v.Name,
		Description: v.Description,
		Lifecycle:   EmoteLifecycleModel(v.State.Lifecycle),
		States:      states.Values(),
		Listed:      v.State.Listed,
		Animated:    v.Animated,
		Host: &ImageHost{
			URL:   fmt.Sprintf("//%s/emote/%s", x.cdnURL, v.ID.Hex()),
			Files: files,
		},
		CreatedAt: v.CreatedAt.UnixMilli(),
	}
}
