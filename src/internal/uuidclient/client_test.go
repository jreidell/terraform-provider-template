package uuidclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jreidell/terraform-provider-template/internal/uuidclient"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		count   int
		status  int
		body    string
		want    int
		wantErr string
	}{
		"success": {
			count:  2,
			status: http.StatusOK,
			body:   `["469b97e8-c724-11f1-a43d-325096b39f47","469b99a0-c724-11f1-acea-325096b39f47"]`,
			want:   2,
		},
		"api error": {
			count:   2,
			status:  http.StatusBadRequest,
			body:    `{"errors":{"count":["The count must be between 1 and 100."]}}`,
			wantErr: "unexpected status 400",
		},
		"malformed body": {
			count:   1,
			status:  http.StatusOK,
			body:    `not json`,
			wantErr: "parsing response body",
		},
		"short response": {
			count:   3,
			status:  http.StatusOK,
			body:    `["469b97e8-c724-11f1-a43d-325096b39f47"]`,
			wantErr: "requested 3 UUIDs but received 1",
		},
		"count too small": {
			count:   0,
			wantErr: "count must be between 1 and 100",
		},
		"count too large": {
			count:   101,
			wantErr: "count must be between 1 and 100",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			// The trailing slash checks that endpoints are normalized.
			client := uuidclient.New(server.URL+"/", server.Client(), "test")

			got, err := client.Generate(context.Background(), tc.count)

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != tc.want {
				t.Fatalf("expected %d UUIDs, got %d", tc.want, len(got))
			}
		})
	}
}

func TestGenerateRequestPath(t *testing.T) {
	t.Parallel()

	var gotPath, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotUserAgent = r.UserAgent()
		_, _ = w.Write([]byte(`["469b97e8-c724-11f1-a43d-325096b39f47"]`))
	}))
	defer server.Close()

	client := uuidclient.New(server.URL, server.Client(), "terraform-provider-template/test")

	if _, err := client.Generate(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPath != "/count/1" {
		t.Errorf("expected request path /count/1, got %s", gotPath)
	}

	if gotUserAgent != "terraform-provider-template/test" {
		t.Errorf("unexpected user agent %q", gotUserAgent)
	}
}
