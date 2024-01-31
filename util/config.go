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
