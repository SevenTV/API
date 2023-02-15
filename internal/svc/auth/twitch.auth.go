package auth

import (
	"encoding/json"

	"github.com/nicklaw5/helix"
)

var twitchScopes = []string{
	"user:read:email",
}

func (a *authorizer) TwichUserData(grant string) (string, []byte, error) {
	client, err := a.helixFactory()
	if err != nil {
		return "", nil, err
	}

	client.SetUserAccessToken(grant)

	resp, err := client.GetUsers(&helix.UsersParams{})
	if err != nil {
		return "", nil, err
	}

	var data helix.User

	if len(resp.Data.Users) > 0 {
		data = resp.Data.Users[0]
	}

	b, err := json.Marshal(data)

	return data.ID, b, err
}
