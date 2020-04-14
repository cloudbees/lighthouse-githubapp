package hook

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cloudbees/jx-tenant-service/pkg/access"
	"github.com/cloudbees/lighthouse-githubapp/pkg/tenant"
	"github.com/jenkins-x/go-scm/scm"
)

func TestWebhooks(t *testing.T) {
	t.Parallel()

	os.Setenv("GO_SCM_LOG_WEBHOOKS", "true")
	insecure := false
	tests := []struct {
		name             string
		event            string
		before           string
		multipleAttempts bool
		workspace        *access.WorkspaceAccess
		handlerFunc      func(rw http.ResponseWriter, req *http.Request)
	}{
		// push
		{
			event:            "push",
			before:           "testdata/push.json",
			multipleAttempts: false,
			workspace:        &access.WorkspaceAccess{Project: "cbjx-mycluster", Cluster: "mycluster", LighthouseURL: "http://dummy-lighthouse-url/hook", HMAC: "MTIzNA==", Insecure: &insecure},
			handlerFunc: func(rw http.ResponseWriter, req *http.Request) {
				// Test request parameters
				assert.Equal(t, req.URL.String(), "/")

				assert.Equal(t, req.Header.Get("X-Hub-Signature"), "sha1=cedda785b4dd580b72f0d4fa92a9697125372c15")
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
		// unknown repository
		{
			name:             "unknown repository",
			event:            "push",
			before:           "testdata/push.json",
			multipleAttempts: true,
			workspace:        &access.WorkspaceAccess{Project: "cbjx-mycluster", Cluster: "mycluster", LighthouseURL: "http://dummy-lighthouse-url/hook", HMAC: "MTIzNA==", Insecure: &insecure},
			handlerFunc: func(rw http.ResponseWriter, req *http.Request) {
				// Test request parameters
				assert.Equal(t, req.URL.String(), "/")

				assert.Equal(t, req.Header.Get("X-Hub-Signature"), "sha1=cedda785b4dd580b72f0d4fa92a9697125372c15")
				// Send response to be tested
				rw.WriteHeader(500)
				_, err := rw.Write([]byte(repoNotConfiguredMessage))
				assert.NoError(t, err)
			},
		},
		// any other error
		{
			name:             "any other error",
			event:            "push",
			before:           "testdata/push.json",
			multipleAttempts: true,
			workspace:        &access.WorkspaceAccess{Project: "cbjx-mycluster", Cluster: "mycluster", LighthouseURL: "http://dummy-lighthouse-url/hook", HMAC: "MTIzNA==", Insecure: &insecure},
			handlerFunc: func(rw http.ResponseWriter, req *http.Request) {
				// Test request parameters
				assert.Equal(t, req.URL.String(), "/")

				assert.Equal(t, req.Header.Get("X-Hub-Signature"), "sha1=cedda785b4dd580b72f0d4fa92a9697125372c15")
				// Send response to be tested
				rw.WriteHeader(500)
				_, err := rw.Write([]byte(`not ok`))
				assert.NoError(t, err)
			},
		},
	}

	for _, test := range tests {
		if test.name == "" {
			test.name = test.before
		}
		t.Run(test.name, func(t *testing.T) {
			before, err := ioutil.ReadFile(test.before)
			if err != nil {
				t.Error(err)
			}

			buf := bytes.NewBuffer(before)
			r, _ := http.NewRequest("POST", "/", buf)
			r.Header.Set("X-GitHub-Event", test.event)
			r.Header.Set("X-GitHub-Delivery", "f2467dea-70d6-11e8-8955-3c83993e0aef")
			r.Header.Set("X-Hub-Signature", "sha1=e9c4409d39729236fda483f22e7fb7513e5cd273")

			retryDuration := 5 * time.Second
			handler := HookOptions{
				tenantService: tenant.NewFakeTenantService(test.workspace),
				secretFn: func(scm.Webhook) (string, error) {
					return "", nil
				},
				maxRetryDuration: &retryDuration,
			}

			attempts := 0
			if test.workspace != nil {
				hf := func(rw http.ResponseWriter, req *http.Request) {
					attempts++
					test.handlerFunc(rw, req)
				}
				server := httptest.NewServer(http.HandlerFunc(hf))
				// Close the server when test finishes
				defer server.Close()
				test.workspace.LighthouseURL = server.URL
				handler.client = server.Client()
			}

			w := NewFakeRespone(t)
			handler.handleWebHookRequests(w, r)

			assert.Equal(t, string(w.body), "OK")
			if test.workspace != nil {
				assert.Equal(t, test.multipleAttempts, attempts > 1, "expected multiple attempts %t, but got %d", test.multipleAttempts, attempts)
			}

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
