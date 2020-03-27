/*
 * The MIT License
 *
 * Copyright (c) 2020, CloudBees, Inc.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package hook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
)

func TestHandleInstalledRequests_Routing(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		repo           string
		expectedStatus int
	}{
		{
			name:           "owner and repo",
			owner:          "foo",
			repo:           "bar",
			expectedStatus: 200,
		},
		{
			name:           "owner, no repo",
			owner:          "foo",
			expectedStatus: 200,
		},
		{
			name:           "no owner, no repo",
			expectedStatus: 404,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			router := muxtrace.NewRouter()

			options := HookOptions{
				githubApp: &testGhaClient{},
			}

			options.Handle(router)

			reqUrl := "/installed/"
			if tc.owner != "" {
				reqUrl += tc.owner + "/"
			}
			if tc.repo != "" {
				reqUrl += tc.repo
			}
			ownerAndRepoReq, err := http.NewRequest("GET", reqUrl, nil)
			if err != nil {
				t.Fatalf("Did not expect an error creating request: %v", err)
			}

			router.ServeHTTP(rr, ownerAndRepoReq)

			assert.Equal(t, tc.expectedStatus, rr.Code)

			if tc.expectedStatus == http.StatusOK {
				body := rr.Body.Bytes()
				resp := &testResponse{}
				err = json.Unmarshal(body, resp)
				assert.NoError(t, err)

				assert.Equal(t, tc.owner, resp.Owner)
				assert.Equal(t, tc.repo, resp.Repo)
			}
		})
	}
}

type testGhaClient struct{}

type testResponse struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

func (c *testGhaClient) handleInstalledRequests(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resp := &testResponse{}
	resp.Owner = vars["owner"]
	resp.Repo = vars["repository"]

	res, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
