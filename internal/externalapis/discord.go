package externalapis

import (
	"fmt"
	"io"
	"net/http"

	"github.com/seventv/api/internal/global"
	"github.com/seventv/common/structures/v3"
)

type discord struct{}

var Discord = discord{}

func (discord) GetCurrentUser(gctx global.Context, token string) (structures.UserConnectionDataDiscord, error) {
	result := structures.UserConnectionDataDiscord{}

	req, err := Discord.DiscordAPIRequest(gctx, "GET", "/users/@me", "")
	if err != nil {
		return result, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return result, err
		}

		return result, fmt.Errorf("bad resp from discord: %d - %s", resp.StatusCode, body)
	}

	if err = ReadRequestResponse(resp, &result); err != nil {
		return result, err
	}

	return result, nil
}
