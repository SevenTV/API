package externalapis

import (
	"fmt"
	"io"
	"net/http"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/structures/v3"
)

type twitch struct{}

var Twitch = twitch{}

func (twitch) GetUserFromToken(gCtx global.Context, token string) ([]structures.UserConnectionDataTwitch, error) {
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

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("bad resp from twitch: %d - %s", resp.StatusCode, body)
	}

	var res getTwitchUsersResp
	if err = ReadRequestResponse(resp, &res); err != nil {
		return nil, err
	}

	return res.Users, nil
}

type GetTwitchUsersParams struct {
	ID    string `url:"id"`
	Login string `url:"login"`
}

type getTwitchUsersResp struct {
	Users []structures.UserConnectionDataTwitch `json:"data"`
}
