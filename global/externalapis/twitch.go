package externalapis

import (
	"fmt"
	"net/http"

	"github.com/SevenTV/Common/structures/v3"
	"github.com/seventv/api/global"
)

type twitch struct{}

var Twitch = twitch{}

func (twitch) GetUsers(gCtx global.Context, token string) ([]*structures.UserConnectionDataTwitch, error) {
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

	var res *TwitchUsersResponse
	if err = ReadRequestResponse(resp, &res); err != nil {
		return nil, err
	}

	return res.Data, nil
}

type GetTwitchUsersParams struct {
	ID    string `url:"id"`
	Login string `url:"login"`
}

type TwitchUsersResponse struct {
	Data []*structures.UserConnectionDataTwitch `json:"data"`
}
