package auth

import "encoding/json"

var discordScopes = []string{
	"identify",
	"email",
}

func (a *authorizer) DiscordUserData(grant string) (string, []byte, error) {
	client, err := a.discordFactory(grant)
	if err != nil {
		return "", nil, err
	}

	user, err := client.User("@me")
	if err != nil {
		return "", nil, err
	}

	b, err := json.Marshal(user)

	return user.ID, b, err
}
