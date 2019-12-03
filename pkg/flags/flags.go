package flags

var (
	// GitHubAppID the ID of the GitHub App
	GitHubAppID = NewIntFlag(0, "LHA_APP_ID")

	// AppPrivateKeyFile the file name for the private key
	AppPrivateKeyFile = NewStringFlag("", "LHA_PRIVATE_KEY_FILE")

	// HmacToken the webhook secret
	HmacToken = NewStringFlag("", "LHA_HMAC_TOKEN")

	// BotName name of the bot
	BotName = NewStringFlag("jenkins-x-bot", "BOT_NAME")

	// HttpPort the port to consumer on
	HttpPort = NewStringFlag("8080", "LHA_HTTP_PORT")

	// GitKind the git server kind
	GitKind = NewStringFlag("github", "LHA_GIT_KIND")

	// GitServer the git server
	GitServer = NewStringFlag("https://github.com", "LHA_GIT_SERVER")

	// GitToken the git token
	GitToken = NewStringFlag("", "LHA_GIT_TOKEN")

	// DebugLogging should we use debug level logging
	DebugLogging = NewBoolFlag(false, "DEBUG_LOGGING")

	// DataDogEnabled should we enable the Datadog tracing
	DataDogEnabled = NewBoolFlag(false, "DD_ENABLED")
)
