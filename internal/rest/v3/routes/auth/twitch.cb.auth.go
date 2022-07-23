package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
)

// twitchCallback:
type twitchCallback struct {
	Ctx global.Context
}

func newTwitchCallback(gCtx global.Context) rest.Route {
	return &twitchCallback{gCtx}
}

func (r *twitchCallback) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:        "/twitch/callback",
		Method:     rest.GET,
		Children:   []rest.Route{},
		Middleware: []rest.Middleware{},
	}
}

func (r *twitchCallback) Handler(ctx *rest.Ctx) rest.APIError {
	// Get state parameter
	state := utils.B2S(ctx.QueryArgs().Peek("state"))
	if state == "" {
		ctx.SetStatusCode(rest.BadRequest)

		return errors.ErrMissingRequiredField().SetFields(errors.Fields{"query": "state"})
	}

	// Retrieve the CSRF token from cookies
	csrfToken := strings.Split(utils.B2S(ctx.Request.Header.Cookie(TWITCH_CSRF_COOKIE_NAME)), ".")
	if len(csrfToken) != 3 {
		ctx.SetStatusCode(rest.BadRequest)

		return errors.ErrUnauthorized().SetDetail(fmt.Sprintf("Bad State (found %d segments when 3 were expected)", len(csrfToken)))
	}

	// Verify the token
	csrfClaim := &auth.JWTClaimOAuth2CSRF{}

	token, err := auth.VerifyJWT(r.Ctx.Config().Credentials.JWTSecret, csrfToken, csrfClaim)
	if err != nil {
		zap.S().Errorw("jwt",
			"error", err,
		)

		ctx.SetStatusCode(rest.BadRequest)

		return errors.ErrUnauthorized().SetDetail(fmt.Sprintf("Invalid State: %s", err.Error()))
	}

	{
		b, err := json.Marshal(token.Claims)
		if err != nil {
			zap.S().Errorw("json",
				"error", err,
			)

			ctx.SetStatusCode(rest.BadRequest)

			return errors.ErrUnauthorized().SetDetail(fmt.Sprintf("Invalid State: %s", err.Error()))
		}

		if err = json.Unmarshal(b, csrfClaim); err != nil {
			zap.S().Errorw("json",
				"error", err,
			)

			ctx.SetStatusCode(rest.BadRequest)

			return errors.ErrUnauthorized().SetDetail(fmt.Sprintf("Invalid State: %s", err.Error()))
		}
	}

	// Validate the token
	// Check date matches
	if csrfClaim.CreatedAt.Before(time.Now().Add(-time.Minute * 5)) {
		return errors.ErrUnauthorized().SetDetail("Expired State")
	}

	// Check token value mismatch
	if state != csrfClaim.State {
		return errors.ErrUnauthorized().SetDetail("Mismatching State Value")
	}

	// Remove the CSRF cookie
	cookie := fasthttp.Cookie{}
	cookie.SetKey(TWITCH_CSRF_COOKIE_NAME)
	cookie.SetExpire(time.Now())
	cookie.SetDomain(r.Ctx.Config().Http.Cookie.Domain)
	cookie.SetSecure(r.Ctx.Config().Http.Cookie.Secure)
	cookie.SetHTTPOnly(true)
	ctx.Response.Header.Cookie(&cookie) // We have now validated this request is authentic.

	// OAuth2 auhorization code for granting an access token
	code := utils.B2S(ctx.QueryArgs().Peek("code"))

	// Format querystring for our authenticated request to twitch
	params, err := query.Values(&OAuth2AuthorizationParams{
		ClientID:     r.Ctx.Config().Platforms.Twitch.ClientID,
		ClientSecret: r.Ctx.Config().Platforms.Twitch.ClientSecret,
		RedirectURI:  r.Ctx.Config().Platforms.Twitch.RedirectURI,
		Code:         code,
		GrantType:    "authorization_code",
	})
	if err != nil {
		zap.S().Errorw("querystring",
			"error", err,
		)

		ctx.SetStatusCode(rest.InternalServerError)

		return errors.ErrInternalServerError()
	}

	// Prepare a HTTP request to Twitch to convert code to acccess token
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://id.twitch.tv/oauth2/token?%s", params.Encode()), nil)
	if err != nil {
		zap.S().Errorw("twitch",
			"error", err,
		)

		ctx.SetStatusCode(rest.InternalServerError)

		return errors.ErrInternalServerError().SetDetail("Internal Request to External Provider Failed")
	}

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		zap.S().Errorw("twitch",
			"error", err,
		)

		return errors.ErrInternalServerError().SetDetail("Internal Request Rejected by External Provider")
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			zap.S().Errorw("twitch",
				"error", err,
			)

			return errors.ErrInternalServerError().SetDetail("Internal Request Rejected by External Provider")
		}

		zap.S().Errorw("twitch",
			"error", fmt.Errorf("bad resp from twitch: %d - %s", resp.StatusCode, body),
		)

		return errors.ErrInternalServerError().SetDetail("Internal Request Rejected by External Provider")
	}

	grant := &OAuth2AuthorizedResponse{}
	if err = externalapis.ReadRequestResponse(resp, grant); err != nil {
		zap.S().Errorw("ReadRequestResponse",
			"error", err,
		)

		ctx.SetStatusCode(rest.InternalServerError)

		return errors.ErrInternalServerError().SetDetail("Failed to decode data sent by the External Provider")
	}

	// Retrieve twitch user data
	users, err := externalapis.Twitch.GetUserFromToken(r.Ctx, grant.AccessToken)
	if err != nil {
		zap.S().Errorw("Twitch, GetUsers",
			"error", err,
		)

		ctx.SetStatusCode(rest.InternalServerError)

		return errors.ErrInternalServerError().SetDetail("Couldn't fetch user data from the External Provider")
	}

	if len(users) == 0 {
		ctx.SetStatusCode(rest.InternalServerError)

		return errors.ErrInternalServerError().SetDetail("No user data response from the External Provider")
	}

	twUser := users[0]
	// Create a new User
	ub := structures.NewUserBuilder(structures.User{
		RoleIDs:     []structures.ObjectID{},
		Editors:     []structures.UserEditor{},
		Connections: []structures.UserConnection[bson.Raw]{},
	})

	usr := structures.UserConnectionDataTwitch{
		ID:              twUser.ID,
		Login:           twUser.Login,
		DisplayName:     twUser.DisplayName,
		BroadcasterType: twUser.BroadcasterType,
		Description:     twUser.Description,
		ProfileImageURL: twUser.ProfileImageURL,
		OfflineImageURL: twUser.OfflineImageURL,
		ViewCount:       twUser.ViewCount,
		Email:           twUser.Email,
		CreatedAt:       twUser.CreatedAt,
	}

	ucb := structures.NewUserConnectionBuilder(structures.UserConnection[structures.UserConnectionDataTwitch]{}).
		SetID(twUser.ID).
		SetPlatform(structures.UserConnectionPlatformTwitch).
		SetLinkedAt(time.Now()).
		SetData(usr).                                                                 // Set twitch data
		SetGrant(grant.AccessToken, grant.RefreshToken, grant.ExpiresIn, grant.Scope) // Update the token grant

	// Write to database
	var userID primitive.ObjectID
	{
		// Find user
		err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOne(ctx, bson.M{
			"connections.id": twUser.ID,
		}).Decode(&ub.User)
		if err == mongo.ErrNoDocuments {
			// User doesn't yet exist: create it
			ucb.UserConnection.EmoteSlots = 250
			ub.SetUsername(twUser.Login).
				SetDisplayName(twUser.DisplayName).
				SetEmail(twUser.Email).
				SetDiscriminator("").
				SetAvatarID("").
				AddConnection(ucb.UserConnection.ToRaw())

			r, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).InsertOne(ctx, ub.User)
			if err != nil {
				zap.S().Errorw("mongo",
					"error", err,
				)
				ctx.SetStatusCode(rest.InternalServerError)

				return errors.ErrInternalServerError().SetDetail("Database Write Failed (user, stat)")
			}

			userID, _ = r.InsertedID.(primitive.ObjectID)
		} else if err != nil {
			zap.S().Errorw("mongo",
				"error", err,
			)

			return errors.ErrInternalServerError().SetDetail("Database Write Failed (user, stat)")
		} else {
			_, pos, _ := ub.User.Connections.Twitch(usr.ID)
			if pos >= 0 {
				ub.Update.Set(fmt.Sprintf("connections.%d.data", pos), usr)
				ub.Update.Set(fmt.Sprintf("connections.%d.grant", pos), grant)
			}

			// User exists; update
			if err = r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameUsers).FindOneAndUpdate(ctx, bson.M{
				"_id":            ub.User.ID,
				"connections.id": usr.ID,
			}, ub.Update, options.FindOneAndUpdate().SetReturnDocument(1)).Decode(&ub.User); err != nil {
				zap.S().Errorw("mongo",
					"error", err,
				)

				return errors.ErrInternalServerError().SetDetail("Database Write Failed (user, stat)")
			}

			userID = ub.User.ID
		}
	}

	// Generate an access token for the user
	tokenTTL := time.Now().Add(time.Hour * 168)

	userToken, err := auth.SignJWT(r.Ctx.Config().Credentials.JWTSecret, &auth.JWTClaimUser{
		UserID:       userID.Hex(),
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
	cookie = fasthttp.Cookie{}
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
	if csrfClaim.OldRedirect {
		websiteURL = r.Ctx.Config().OldWebsiteURL
	}

	ctx.Redirect(fmt.Sprintf("%s/oauth2?%s", websiteURL, params.Encode()), int(rest.Found))

	return nil
}
