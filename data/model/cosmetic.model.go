package model

type CosmeticPaintModel struct {
	ID       string                      `json:"id"`
	Function CosmeticPaintFunction       `json:"function" enums:"LINEAR_GRADIENT,RADIAL_GRADIENT,URL"`
	Color    int32                       `json:"color"`
	Repeat   bool                        `json:"repeat"`
	Angle    int32                       `json:"angle"`
	Shape    string                      `json:"shape"`
	ImageURL string                      `json:"image_url"`
	Stops    []CosmeticPaintGradientStop `json:"stops"`
	Shadows  []CosmeticPaintDropShadow   `json:"shadows"`
}

type CosmeticPaintFunction string

const (
	CosmeticPaintFunctionLinearGradient CosmeticPaintFunction = "LINEAR_GRADIENT"
	CosmeticPaintFunctionRadialGradient CosmeticPaintFunction = "RADIAL_GRADIENT"
	CosmeticPaintFunctionImageURL       CosmeticPaintFunction = "URL"
)

type CosmeticPaintGradientStop struct {
	At    float64 `json:"at"`
	Color int32   `json:"color"`
}

type CosmeticPaintDropShadow struct {
	OffsetX float64 `json:"x_offset"`
	OffsetY float64 `json:"y_offset"`
	Radius  float64 `json:"radius"`
	Color   int32   `json:"color"`
}
