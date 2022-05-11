package model

// An Emote Object
// @Description Represents an Emote
type Emote struct {
	// The emote's ID
	ID string `json:"id" swaggertype:"string"`
	// The emote's name
	Name string `json:"name" swaggertype:"string"`
}
