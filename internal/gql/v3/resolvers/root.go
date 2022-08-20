package resolvers

import (
	"github.com/seventv/api/internal/gql/v3/gen/generated"
	"github.com/seventv/api/internal/gql/v3/resolvers/ban"
	"github.com/seventv/api/internal/gql/v3/resolvers/cosmetics"
	"github.com/seventv/api/internal/gql/v3/resolvers/emote"
	emoteversion "github.com/seventv/api/internal/gql/v3/resolvers/emote-version"
	"github.com/seventv/api/internal/gql/v3/resolvers/emoteset"
	activeemote "github.com/seventv/api/internal/gql/v3/resolvers/emoteset/active-emote"
	"github.com/seventv/api/internal/gql/v3/resolvers/mutation"
	"github.com/seventv/api/internal/gql/v3/resolvers/query"
	"github.com/seventv/api/internal/gql/v3/resolvers/report"
	"github.com/seventv/api/internal/gql/v3/resolvers/role"
	"github.com/seventv/api/internal/gql/v3/resolvers/subscription"
	"github.com/seventv/api/internal/gql/v3/resolvers/user"
	user_editor "github.com/seventv/api/internal/gql/v3/resolvers/user-editor"
	user_emote "github.com/seventv/api/internal/gql/v3/resolvers/user-emote"

	"github.com/seventv/api/internal/gql/v3/types"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.ResolverRoot {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) Ban() generated.BanResolver {
	return ban.New(r.Resolver)
}

func (r *Resolver) Emote() generated.EmoteResolver {
	return emote.New(r.Resolver)
}

func (r *Resolver) EmotePartial() generated.EmotePartialResolver {
	return emote.NewPartial(r.Resolver)
}

func (r *Resolver) CosmeticOps() generated.CosmeticOpsResolver {
	return cosmetics.NewOps(r.Resolver)
}

func (r *Resolver) EmoteOps() generated.EmoteOpsResolver {
	return emote.NewOps(r.Resolver)
}

func (r *Resolver) EmoteVersion() generated.EmoteVersionResolver {
	return emoteversion.New(r.Resolver)
}

func (r *Resolver) Mutation() generated.MutationResolver {
	return mutation.New(r.Resolver)
}

func (r *Resolver) Query() generated.QueryResolver {
	return query.New(r.Resolver)
}

func (r *Resolver) Subscription() generated.SubscriptionResolver {
	return subscription.New(r.Resolver)
}

func (r *Resolver) Report() generated.ReportResolver {
	return report.New(r.Resolver)
}

func (r *Resolver) Role() generated.RoleResolver {
	return role.New(r.Resolver)
}

func (r *Resolver) User() generated.UserResolver {
	return user.New(r.Resolver)
}

func (r *Resolver) UserOps() generated.UserOpsResolver {
	return user.NewOps(r.Resolver)
}

func (r *Resolver) UserEditor() generated.UserEditorResolver {
	return user_editor.New(r.Resolver)
}

func (r *Resolver) UserEmote() generated.UserEmoteResolver {
	return user_emote.New(r.Resolver)
}

func (r *Resolver) EmoteSet() generated.EmoteSetResolver {
	return emoteset.New(r.Resolver)
}

func (r *Resolver) EmoteSetOps() generated.EmoteSetOpsResolver {
	return emoteset.NewOps(r.Resolver)
}

func (r *Resolver) ActiveEmote() generated.ActiveEmoteResolver {
	return activeemote.New(r.Resolver)
}
