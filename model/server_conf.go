package model

// ClientDefaults Defaults for creation of new clients used in the templates
type ThumbnailConf struct {
	PercentPosition int
	Duration        int
}
type ServerConf struct {
	JwtSecret    string
	ThumbnailCfg []ThumbnailConf
}
