package externalapis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/seventv/api/internal/global"
)

var TwitchHelixBase = "https://api.twitch.tv/helix"

var DiscordAPIBase = "https://discord.com/api/v10"

func (twitch) HelixAPIRequest(gctx global.Context, method string, route string, params string) (*http.Request, error) {
	uri := fmt.Sprintf("%s%s", TwitchHelixBase, route)

	return http.NewRequestWithContext(gctx, method, uri, nil)
}

func (discord) DiscordAPIRequest(gctx global.Context, method, route, params string) (*http.Request, error) {
	uri := fmt.Sprintf("%s%s", DiscordAPIBase, route)

	return http.NewRequestWithContext(gctx, method, uri, nil)
}

// ReadRequestResponse: quick utility for decoding an api response to a struct
func ReadRequestResponse(resp *http.Response, out interface{}) error {
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(b, out); err != nil {
		return err
	}

	return nil
}
