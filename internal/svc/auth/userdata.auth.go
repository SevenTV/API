package auth

import "github.com/seventv/common/structures/v3"

func (a *authorizer) UserData(provider structures.UserConnectionPlatform, token string) (id string, b []byte, err error) {
	switch provider {
	case structures.UserConnectionPlatformTwitch:
		id, b, err = a.TwichUserData(token)
	case structures.UserConnectionPlatformDiscord:
		id, b, err = a.DiscordUserData(token)
	case structures.UserConnectionPlatformKick:
		id, b, err = a.KickUserData(token)
	}

	if err != nil {
		return "", nil, err
	}

	return id, b, err
}
