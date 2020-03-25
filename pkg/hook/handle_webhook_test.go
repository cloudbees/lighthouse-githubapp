package hook

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/cloudbees/jx-tenant-service/pkg/access"
	"github.com/cloudbees/lighthouse-githubapp/pkg/tenant"
	"github.com/jenkins-x/go-scm/scm"
)

func TestWebhooks(t *testing.T) {
	t.Parallel()

	os.Setenv("GO_SCM_LOG_WEBHOOKS", "true")
	tests := []struct {
		event       string
		before      string
		workspace   *access.WorkspaceAccess
		handlerFunc func(rw http.ResponseWriter, req *http.Request)
	}{
		// push
		{
			event:     "push",
			before:    "testdata/push.json",
			workspace: &access.WorkspaceAccess{Project: "cbjx-mycluster", Cluster: "mycluster", LighthouseURL: "http://dummy-lighthouse-url/hook", HMAC: "1234"},
			handlerFunc: func(rw http.ResponseWriter, req *http.Request) {
				// Test request parameters
				assert.Equal(t, req.URL.String(), "/")
				// Send response to be tested
				_, err := rw.Write([]byte(`OK`))
				assert.NoError(t, err)
			},
		},
		// installation of GitHub App
		{
			event:  "installation",
			before: "testdata/installation.json",
		},
		// delete installation of GitHub App
		{
			event:  "installation",
			before: "testdata/installation_delete.json",
		},
	}

	for _, test := range tests {
		t.Run(test.before, func(t *testing.T) {
			before, err := ioutil.ReadFile(test.before)
			if err != nil {
				t.Error(err)
			}

			buf := bytes.NewBuffer(before)
			r, _ := http.NewRequest("POST", "/", buf)
			r.Header.Set("X-GitHub-Event", test.event)
			r.Header.Set("X-GitHub-Delivery", "f2467dea-70d6-11e8-8955-3c83993e0aef")
			r.Header.Set("X-Hub-Signature", "sha1=e9c4409d39729236fda483f22e7fb7513e5cd273")

			handler := HookOptions{
				tenantService: tenant.NewFakeTenantService(test.workspace),
				secretFn: func(scm.Webhook) (string, error) {
					return "", nil
				},
			}

			if test.workspace != nil {
				server := httptest.NewServer(http.HandlerFunc(test.handlerFunc))
				// Close the server when test finishes
				defer server.Close()
				test.workspace.LighthouseURL = server.URL
				handler.client = server.Client()
			}

			w := NewFakeRespone(t)
			handler.handleWebHookRequests(w, r)

			assert.Equal(t, string(w.body), "OK")
		})
	}
}

type FakeResponse struct {
	t       *testing.T
	headers http.Header
	body    []byte
	status  int
}

func NewFakeRespone(t *testing.T) *FakeResponse {
	return &FakeResponse{
		t:       t,
		headers: make(http.Header),
	}
}

func (r *FakeResponse) Header() http.Header {
	return r.headers
}

func (r *FakeResponse) Write(body []byte) (int, error) {
	r.body = body
	return len(body), nil
}

func (r *FakeResponse) WriteHeader(status int) {
	r.status = status
}
