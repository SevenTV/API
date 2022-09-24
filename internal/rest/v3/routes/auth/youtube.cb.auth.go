package auth

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-querystring/query"
	"github.com/seventv/api/internal/externalapis"
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/auth"
	"github.com/seventv/common/errors"
	"github.com/seventv/common/mongo"
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	oauthapi "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
	ytapi "google.golang.org/api/youtube/v3"
)

type youtubeCallback struct {
	Ctx global.Context
}

func newYoutubeCallback(gCtx global.Context) rest.Route {
	return &youtubeCallback{gCtx}
}

func (r *youtubeCallback) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:        "/youtube/callback",
		Method:     rest.GET,
		Children:   []rest.Route{},
		Middleware: []rest.Middleware{},
	}
}

func (r *youtubeCallback) Handler(ctx *rest.Ctx) rest.APIError {
	stateToken, err := handleOAuthState(r.Ctx, ctx, YOUTUBE_CSRF_COOKIE_NAME)
	if err != nil {
		return errors.From(err)
	}

	// OAuth2 authorization code
	code := utils.B2S(ctx.QueryArgs().Peek("code"))

	// Format querystring for our request to youtube
	params, err := query.Values(&OAuth2AuthorizationParams{
		ClientID:     r.Ctx.Config().Platforms.YouTube.ClientID,
		ClientSecret: r.Ctx.Config().Platforms.YouTube.ClientSecret,
		RedirectURI:  r.Ctx.Config().Platforms.YouTube.RedirectURI,
		Code:         code,
		GrantType:    "authorization_code",
	})
	if err != nil {
		ctx.Log().Errorw("querystring",
			"error", err,
		)

		return errors.ErrInternalServerError()
	}

	// Set up a request to google to get an access token
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://oauth2.googleapis.com/token?%s", params.Encode()), nil)
	if err != nil {
		ctx.Log().Errorw("youtube", "error", err)
	}

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ctx.Log().Errorw("youtube", "error", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			ctx.Log().Errorw("youtube", "error", err)

			return errors.ErrInternalServerError().SetDetail("Token Exchange Rejected (googleapis.com)")
		}

		ctx.Log().Errorw("youtube",
			"status", resp.StatusCode,
			"error", utils.B2S(body),
		)

		return errors.ErrInternalServerError().SetDetail("Non-OK response from googleapis.com")
	}

	grant := &oauth2.Token{}
	if err = externalapis.ReadRequestResponse(resp, grant); err != nil {
		ctx.Log().Errorw("ReadRequestResponse", "error", err)

		return errors.ErrInternalServerError().SetDetail("Failed to decode data sent by googleapis.com")
	}

	tokenSource := option.WithTokenSource(oauth2.StaticTokenSource(grant))

	gsvc, err := oauthapi.NewService(ctx, tokenSource)
	if err != nil {
		ctx.Log().Errorw("unable to setup oauth instance", "error", err)

		return errors.ErrInternalServerError().SetDetail("Failed to create oauth2 service")
	}

	uinfo, err := gsvc.Userinfo.Get().Do()
	if err != nil {
		ctx.Log().Errorw("unable to get user info", "error", err)

		return errors.ErrInternalServerError().SetDetail("Failed to get user info from googleapis.com")
	}

	// Fetch youtube channel data
	ytsvc, err := ytapi.NewService(ctx, tokenSource)
	if err != nil {
		ctx.Log().Errorw("unable to setup youtube instance with bearer token", "error", err)

		return errors.ErrInternalServerError().SetDetail("Failed to authenticate with YouTube")
	}

	channels, err := ytsvc.Channels.List([]string{"snippet", "statistics"}).Mine(true).Do()
	if err != nil {
		ctx.Log().Errorw("unable to get youtube channel info", "error", err)

		return errors.ErrInternalServerError().SetDetail("Failed to get channel info from googleapis.com")
	}

	ids := make([]string, len(channels.Items))
	ytList := make([]structures.UserConnectionDataYoutube, len(channels.Items))
	resultRaw := make([]bson.Raw, len(channels.Items))

	for i, channel := range channels.Items {
		ytList[i] = structures.UserConnectionDataYoutube{
			ID:              channel.Id,
			Title:           channel.Snippet.Title,
			Description:     channel.Snippet.Description,
			ProfileImageURL: channel.Snippet.Thumbnails.High.Url,
			SubCount:        int64(channel.Statistics.SubscriberCount),
			ViewCount:       int64(channel.Statistics.ViewCount),
		}

		resultRaw[i], _ = bson.Marshal(ytList[i])

		ids[i] = channel.Id
	}

	if len(ytList) == 0 {
		return errors.ErrInternalServerError().SetDetail("Your google account does not have any youtube channels")
	}

	firstChannel := ytList[0]

	ucb := structures.NewUserConnectionBuilder(structures.UserConnection[structures.UserConnectionDataYoutube]{
		ChoiceData: resultRaw,
		EmoteSlots: 250,
	}).
		SetID(firstChannel.ID).
		SetPlatform(structures.UserConnectionPlatformYouTube).
		SetLinkedAt(time.Now()).
		SetData(firstChannel).
		SetGrant(grant.AccessToken, grant.RefreshToken, int(time.Since(grant.Expiry).Seconds()), youtubeScopes)

	// Find the user
	f := utils.Ternary(stateToken.Bind.IsZero(), bson.M{
		"connections.id": bson.M{"$in": ids},
	}, bson.M{
		"_id": stateToken.Bind,
	})

	ub := structures.NewUserBuilder(structures.User{
		ID:          primitive.NewObjectIDFromTimestamp(time.Now()),
		RoleIDs:     []primitive.ObjectID{},
		Editors:     []structures.UserEditor{},
		Connections: []structures.UserConnection[bson.Raw]{},
	})

	err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOne(ctx, f).Decode(&ub.User)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			ctx.Log().Errorw("error occured trying to find user for youtube connection", "error", err)

			return errors.ErrInternalServerError()
		}

		// No user with this connection exists, s o we'll create a new one

		// Create username for this user
		username := strings.Builder{}

		title := ytUsernameRegexp.ReplaceAllString(firstChannel.Title, "")
		title = title[:int(math.Min(21, float64(len(title))))]

		_, _ = username.WriteString(title)
		_, _ = username.WriteString("@YT")

		ub.SetUsername(strings.ToLower(username.String())).
			SetDisplayName(username.String()).
			SetEmail(uinfo.Email).
			SetDiscriminator("").
			SetAvatarID("").
			AddConnection(ucb.UserConnection.ToRaw())

		// Write the user to database
		_, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).InsertOne(ctx, ub.User)
		if err != nil {
			ctx.Log().Errorw("error occured trying to write new user after youtube connection", "error", err)

			return errors.ErrInternalServerError().SetDetail("Database Write Failed (user, stat)")
		}
	} else { // User exists, so we'll update their connection
		for _, ch := range ytList {
			_, pos, _ := ub.User.Connections.YouTube(ch.ID)
			if pos == -1 {
				// new connection
				ub.AddConnection(ucb.UserConnection.ToRaw())

				continue
			}

			ub.Update.Set(fmt.Sprintf("connections.%d.data", pos), ch)
			ub.Update.Set(fmt.Sprintf("connections.%d.grant", pos), structures.UserConnectionGrant{
				AccessToken:  grant.AccessToken,
				RefreshToken: grant.RefreshToken,
				ExpiresAt:    grant.Expiry,
			})
		}

		// User exists; update
		if err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOneAndUpdate(ctx, bson.M{
			"_id": ub.User.ID,
		}, ub.Update, options.FindOneAndUpdate().SetReturnDocument(1)).Decode(&ub.User); err != nil {
			ctx.Log().Errorw("mongo",
				"error", err,
			)

			return errors.ErrInternalServerError().SetDetail("Database Write Failed (user, stat)")
		}
	}

	// Generate an access token for the user
	tokenTTL := time.Now().Add(time.Hour * 2190)

	userToken, err := auth.SignJWT(r.Ctx.Config().Credentials.JWTSecret, &auth.JWTClaimUser{
		UserID:       ub.User.ID.Hex(),
		TokenVersion: ub.User.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "7TV-API-REST",
			ExpiresAt: &jwt.NumericDate{
				Time: tokenTTL,
			},
		},
	})
	if err != nil {
		zap.S().Errorw("jwt",
			"error", err,
		)

		return errors.ErrInternalServerError().SetDetail(fmt.Sprintf("Token Sign Failure (%s)", err.Error()))
	}

	// Define a cookie
	cookie := fasthttp.Cookie{}
	cookie.SetKey("access_token")
	cookie.SetValue(userToken)
	cookie.SetDomain(r.Ctx.Config().Http.Cookie.Domain)
	cookie.SetSecure(r.Ctx.Config().Http.Cookie.Secure)
	cookie.SetHTTPOnly(true)
	ctx.Response.Header.Cookie(&cookie)

	// Redirect to website's callback page
	params, _ = query.Values(&OAuth2CallbackAppParams{
		Token: userToken,
	})

	websiteURL := r.Ctx.Config().WebsiteURL
	if stateToken.OldRedirect {
		websiteURL = r.Ctx.Config().OldWebsiteURL
	}

	ctx.Redirect(fmt.Sprintf("%s/oauth2?%s", websiteURL, params.Encode()), int(rest.Found))

	return nil
}

var ytUsernameRegexp = regexp.MustCompile("[^a-zA-Z0-9]+")
