package flags

var (
	// HmacToken the webhook secret
	HmacToken = NewStringFlag("", "LHA_HMAC_TOKEN")

	// HttpPort the port to consumer on
	HttpPort = NewStringFlag("8080", "LHA_HTTP_PORT")

	// GitKind the git server kind
	GitKind = NewStringFlag("github", "LHA_GIT_KIND")

	// GitServer the git server
	GitServer = NewStringFlag("https://github.com", "LHA_GIT_SERVER")

	// GitToken the git token
	GitToken = NewStringFlag("TODO", "LHA_GIT_TOKEN")
)
