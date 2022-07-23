package auth

import (
	"github.com/seventv/api/internal/global"
	"github.com/seventv/api/internal/rest/rest"
	"github.com/seventv/common/errors"
)

type Route struct {
	Ctx global.Context
}

func New(gCtx global.Context) rest.Route {
	return &Route{gCtx}
}

func (r *Route) Config() rest.RouteConfig {
	return rest.RouteConfig{
		URI:    "/auth",
		Method: rest.GET,
		Children: []rest.Route{
			newTwitch(r.Ctx),
			newTwitchCallback(r.Ctx),
			newDiscord(r.Ctx),
			newDiscordCallback(r.Ctx),
		},
		Middleware: []rest.Middleware{},
	}
}

func (r *Route) Handler(ctx *rest.Ctx) rest.APIError {
	return errors.ErrInvalidRequest().WithHTTPStatus(int(rest.SeeOther)).SetDetail("Use OAuth2 routes")
}

type OAuth2URLParams struct {
	ClientID     string `url:"client_id"`
	RedirectURI  string `url:"redirect_uri"`
	ResponseType string `url:"response_type"`
	Scope        string `url:"scope"`
	State        string `url:"state"`
}

type OAuth2AuthorizationParams struct {
	ClientID     string `url:"client_id" json:"client_id"`
	ClientSecret string `url:"client_secret" json:"client_secret"`
	RedirectURI  string `url:"redirect_uri" json:"redirect_uri"`
	Code         string `url:"code" json:"code"`
	GrantType    string `url:"grant_type" json:"grant_type"`
	Scope        string `url:"scope" json:"scope"`
}

type OAuth2AuthorizedResponse struct {
	TokenType    string   `json:"token_type"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	Scope        []string `json:"scope"`
	ExpiresIn    int      `json:"expires_in"`
}

type OAuth2AuthorizedResponseAlt struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
}
