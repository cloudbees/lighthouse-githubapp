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

	// GithubApp path query endpoint to determine if repository is installed for a github app
	GithubAppPath = "/installed/{owner}/{repository}"

	// TestTokenPath tests the token for the given github app owner installation
	TestTokenPath = "/debug/ghtoken/{installation}"

	// tokenCacheExpiration how long should the tokens be cached for
	tokenCacheExpiration = 10 * time.Minute

	// tokenCacheExpireDelta the delta before the expire time when we re-request a token to ensure we don't get too close to the limit
	tokenCacheExpireDelta = 5 * time.Minute

	// to verify scm tokens
	verifyRepository  = "arcalos-environments/environment-devourerpaint-staging"
	verifyPullRequest = 9
)
