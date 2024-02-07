package util

import (
	"journal/model"
)

// Runtime config
var (
	BindAddress  string
	JwtSecret    string
	BasePath     string
	ThumbnailCfg []model.ThumbnailConf
)

const (
	DefaultUsername = "admin"
	DefaultPassword = "123456"
	DefaultIsAdmin  = true
)

var (
	DefaultThumbnailCfg = []model.ThumbnailConf{
		{PercentPosition: 5, Duration: 2},
		{PercentPosition: 25, Duration: 2},
		{PercentPosition: 45, Duration: 2},
	}
)
