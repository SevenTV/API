package model

import (
	"github.com/seventv/common/structures/v3"
	"github.com/seventv/common/utils"
)

type User struct {
	ID          string `json:"id"`
	TwitchID    string `json:"twitch_id"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	Role        *Role  `json:"role"`
}

func NewUser(s structures.User) *User {
	tw, _, _ := s.Connections.Twitch()

	u := User{
		ID:          s.ID.Hex(),
		Login:       s.Username,
		DisplayName: utils.Ternary(s.DisplayName != "", s.DisplayName, s.Username),
		Role:        NewRole(s.GetHighestRole()),
		TwitchID:    tw.ID,
	}

	return &u
}
