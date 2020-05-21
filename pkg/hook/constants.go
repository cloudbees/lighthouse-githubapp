package hook

import "time"

const (
	// SetupPath URL path for the HTTP endpoint for the setup page
	SetupPath = "/setup"
	// HookPath URL path for the HTTP endpoint for handling webhooks
	HookPath = "/hook"
	// HealthPath is the URL path for the HTTP endpoint that returns health status.
	HealthPath = "/health"
	// ReadyPath URL path for the HTTP endpoint that returns ready status.
	ReadyPath = "/ready"

	// GitHubAppPathWithoutRepository path query endpoint for cases where no repository is specified
	GitHubAppPathWithoutRepository = "/installed/{owner}/"

	// GithubApp path query endpoint to determine if repository is installed for a github app
	GithubAppPath = "/installed/{owner}/{repository}"

	// tokenCacheExpiration how long should the tokens be cached for
	tokenCacheExpiration = 10 * time.Minute
)
