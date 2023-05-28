package auth

import (
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"

	"github.com/seventv/common/structures/v3"
)

type KickUserData struct {
	ID     int    `json:"id"`
	UserID int    `json:"user_id"`
	Slug   string `json:"slug"`
	User   struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Bio      string `json:"bio"`
	} `json:"user"`
	Chatroom struct {
		ID int `json:"id"`
	} `json:"chatroom"`
}

func newKickClient(ctx context.Context, token string) *http.Client {
	return &http.Client{}
}

func (a *authorizer) KickUserData(slug string) (string, []byte, error) {
	if a.kickClient == nil {
		return "", nil, nil
	}

	n := rand.Int31()

	res, err := a.kickClient.Do(&http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "https", Host: "kick.com", Path: "/api/v2/channels/" + slug, RawQuery: "7tv-bust" + strconv.Itoa(int(n))},
		Header: http.Header{
			"User-Agent":   {"SevenTV-API/3"},
			"Content-Type": {"application/json"},
			"x-kick-auth":  {a.Config.Kick.ChallengeToken},
		},
	})
	if err != nil {
		return "", nil, err
	}

	// read body to bytes
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", nil, err
	}

	u := KickUserData{}
	if err = json.Unmarshal(b, &u); err != nil {
		return "", nil, err
	}

	connData := structures.UserConnectionDataKick{
		ID:          strconv.Itoa(u.UserID),
		ChatroomID:  strconv.Itoa(u.Chatroom.ID),
		Username:    u.Slug,
		DisplayName: u.User.Username,
		Bio:         u.User.Bio,
	}

	b, err = json.Marshal(connData)

	return connData.ID, b, err
}
