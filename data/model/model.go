package model

import "github.com/seventv/common/structures/v3"

type Modelizer interface {
	User(v structures.User) UserModel
}

type modelizer struct {
	cdnURL     string
	websiteURL string
}

func NewInstance(opt ModelInstanceOptions) Modelizer {
	return &modelizer{
		cdnURL:     opt.CDN,
		websiteURL: opt.Website,
	}
}

type ModelInstanceOptions struct {
	CDN     string
	Website string
}
