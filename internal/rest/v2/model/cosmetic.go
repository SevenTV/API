package model

type CosmeticsMap struct {
	Badges []*CosmeticBadge `json:"badges"`
	Paints []*CosmeticPaint `json:"paints"`
}

type CosmeticBadge struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Tooltip string      `json:"tooltip"`
	URLs    [][2]string `json:"urls"`
	Users   []string    `json:"users"`
	Misc    bool        `json:"misc,omitempty"`
}

type CosmeticPaint struct {
	ID          string                      `json:"id"`
	Name        string                      `json:"name"`
	Users       []string                    `json:"users"`
	Function    string                      `json:"function"`
	Color       *int32                      `json:"color"`
	Stops       []CosmeticPaintGradientStop `json:"stops"`
	Repeat      bool                        `json:"repeat"`
	Angle       int32                       `json:"angle"`
	Shape       string                      `json:"shape,omitempty"`
	ImageURL    string                      `json:"image_url,omitempty"`
	DropShadows []CosmeticPaintDropShadow   `json:"drop_shadows,omitempty"`
}

type CosmeticPaintGradientStop struct {
	At    float64 `json:"at" bson:"at"`
	Color int32   `json:"color" bson:"color"`
}

type CosmeticPaintDropShadow struct {
	OffsetX float64 `json:"x_offset" bson:"x_offset"`
	OffsetY float64 `json:"y_offset" bson:"y_offset"`
	Radius  float64 `json:"radius" bson:"radius"`
	Color   int32   `json:"color" bson:"color"`
}
