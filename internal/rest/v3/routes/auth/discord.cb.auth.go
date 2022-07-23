package auth

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-querystring/query"
	"github.com/seventv/api/internal/events"
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
)

type discordCallback struct {
	Ctx global.Context
}

func newDiscordCallback(gctx global.Context) rest.Route {
	return &discordCallback{gctx}
}

func (r *discordCallback) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:        "/discord/callback",
		Method:     rest.GET,
		Children:   []rest.Route{},
		Middleware: []rest.Middleware{},
	}
}

func (r *discordCallback) Handler(ctx *rest.Ctx) rest.APIError {
	stateToken, err := handleOAuthState(r.Ctx, ctx)
	if err != nil {
		return errors.From(err)
	}

	// OAuth2 auhorization code for granting an access token
	code := utils.B2S(ctx.QueryArgs().Peek("code"))

	// Format querystring for our authenticated request to discord
	params, err := query.Values(&OAuth2AuthorizationParams{
		ClientID:     r.Ctx.Config().Platforms.Discord.ClientID,
		ClientSecret: r.Ctx.Config().Platforms.Discord.ClientSecret,
		RedirectURI:  r.Ctx.Config().Platforms.Discord.RedirectURI,
		Code:         code,
		GrantType:    "authorization_code",
	})
	if err != nil {
		ctx.Log().Errorw("querystring", "error", err)

		return errors.ErrInternalServerError()
	}

	// Now we will make a request to Discord to retrieve the user data
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/oauth2/token", externalapis.DiscordAPIBase), bytes.NewBuffer(utils.S2B(params.Encode())))
	if err != nil {
		ctx.Log().Errorw("discord", "error", err)

		return errors.ErrInternalServerError().SetDetail("Internal Request to External Provider Failed")
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ctx.Log().Errorw("discord", "error", err)

		return errors.ErrInternalServerError().SetDetail("Internal Request Rejected by External Provider")
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			ctx.Log().Errorw("discord", "error", err)

			return errors.ErrInternalServerError().SetDetail("Internal Request Rejected by External Provider")
		}

		ctx.Log().Errorw("discord, bad resp", "error", string(body), "status", resp.StatusCode, "data", string(body))

		return errors.ErrInternalServerError().SetDetail("Internal Request Rejected by External Provider")
	}

	grant := &OAuth2AuthorizedResponse[string]{}
	if err = externalapis.ReadRequestResponse(resp, grant); err != nil {
		ctx.Log().Errorw("ReadRequestResponse",
			"error", err,
		)

		return errors.ErrInternalServerError().SetDetail("Failed to decode data sent by the External Provider")
	}

	// Retrieve discord user data
	diUser, err := externalapis.Discord.GetCurrentUser(r.Ctx, grant.AccessToken)
	if err != nil {
		ctx.Log().Errorw("discord, GetCurrentUser", "error", err)

		return errors.ErrInternalServerError().SetDetail("Couldn't fetch data from the external provider")
	}

	// Set up a user
	ub := structures.NewUserBuilder(structures.User{
		RoleIDs:     []structures.ObjectID{},
		Editors:     []structures.UserEditor{},
		Connections: structures.UserConnectionList{},
	})

	// Add the user data to db and complete the connection
	ucb := structures.NewUserConnectionBuilder(structures.UserConnection[structures.UserConnectionDataDiscord]{}).
		SetID(diUser.ID).
		SetPlatform(structures.UserConnectionPlatformDiscord).
		SetLinkedAt(time.Now()).
		SetData(diUser).
		SetGrant(grant.AccessToken, grant.RefreshToken, grant.ExpiresIn, strings.Split(grant.Scope, " "))

	// Find user
	if err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOne(ctx, bson.M{"$or": bson.A{
		bson.M{"connections.id": diUser.ID},
		bson.M{"_id": stateToken.Bind},
	}}).Decode(&ub.User); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.ErrUnknownUser() // TODO: allow creating an account from a Discord sign in
		}

		ctx.Log().Errorw("mongo", "error", err)

		return errors.ErrInternalServerError()
	}

	// Get connectionn
	_, pos, _ := ub.User.Connections.Discord(diUser.ID)
	if pos >= 0 {
		ub.Update.Set(fmt.Sprintf("connections.%d.data", pos), diUser)
		ub.Update.Set(fmt.Sprintf("connections.%d.grant", pos), structures.UserConnectionGrant{
			AccessToken:  grant.TokenType,
			RefreshToken: grant.AccessToken,
			Scope:        strings.Split(grant.Scope, " "),
			ExpiresAt:    time.Now().Add(time.Second * time.Duration(grant.ExpiresIn)),
		})
	} else { // connection doesn't exist yet, we may append it
		ucb.UserConnection.EmoteSlots = 0

		ub.AddConnection(ucb.UserConnection.ToRaw())
	}

	// Update the user
	if _, err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).UpdateOne(ctx, bson.M{
		"_id": ub.User.ID,
	}, ub.Update); err != nil {
		ctx.Log().Errorw("mongo", "error", err)

		return errors.ErrInternalServerError().SetDetail("Write to database failed")
	}

	// Generate an access token for the user
	tokenTTL := time.Now().Add(time.Hour * 168)

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
		ctx.Log().Errorw("jwt", "error", err)

		return errors.ErrInternalServerError().SetDetail("Token Sign Failure (%s)", err.Error())
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

	ctx.Redirect(fmt.Sprintf("%s/oauth2?%s", r.Ctx.Config().WebsiteURL, params.Encode()), int(rest.Found))

	// Publish user update
	events.Publish(r.Ctx, "users", ub.User.ID)

	// TODO: Request role sync with discord

	return nil
}
