package hook

import (
	"github.com/cloudbees/jx-tenant-service/pkg/access"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestCanDetermineInsecureWebhooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		url            string
		insecure           bool
	}{
		{
			name:     "play-environment",
			url:      "https://hook-jx.cbjx-lynxquiver.play-jxaas.live/hook",
			insecure: true,
		},
		{
			name:     "staging-environment",
			url:      "https://hook-jx.cbjx-lynxquiver.staging-jxaas.live/hook",
			insecure: true,
		},
		{
			name:     "prod-environment",
			url:      "https://hook-jx.cbjx-lynxquiver.jxaas.live/hook",
			insecure: false,
		},
		{
			name:     "pr-environment",
			url:      "https://hook-jx.cbjx-pr-943-1arc.staging-jxaas.live/hook",
			insecure: true,
		},
		{
			name:     "default-environment",
			url:      "https://otherrandomurl.com/h",
			insecure: false,
		},
	}


	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ws := &access.WorkspaceAccess{ LighthouseURL: test.url }
			assert.Equal(t, test.insecure, ShouldUseInsecureRelay(ws))
		})
	}
}