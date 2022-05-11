package externalapis

import (
	"fmt"
	"net/http"

	"github.com/nicklaw5/helix"
	"github.com/seventv/api/internal/global"
)

type twitch struct{}

var Twitch = twitch{}

func (twitch) GetUserFromToken(gCtx global.Context, token string) ([]helix.User, error) {
	req, err := Twitch.HelixAPIRequest(gCtx, "GET", "/users", "")
	if err != nil {
		return nil, err
	}
	req.Header.Add("Client-Id", gCtx.Config().Platforms.Twitch.ClientID)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res helix.ManyUsers
	if err = ReadRequestResponse(resp, &res); err != nil {
		return nil, err
	}

	return res.Users, nil
}

type GetTwitchUsersParams struct {
	ID    string `url:"id"`
	Login string `url:"login"`
}
