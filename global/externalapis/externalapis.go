package externalapis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/seventv/api/global"
)

var TwitchHelixBase = "https://api.twitch.tv/helix"

func (twitch) HelixAPIRequest(gCtx global.Context, method string, route string, params string) (*http.Request, error) {
	uri := fmt.Sprintf("%s%s", TwitchHelixBase, route)

	return http.NewRequestWithContext(gCtx, method, uri, nil)
}

// ReadRequestResponse: quick utility for decoding an api response to a struct
func ReadRequestResponse(resp *http.Response, out interface{}) error {
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(b, &out); err != nil {
		return err
	}

	return nil
}
