package authz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)


func fakeGitHubServer(t *testing.T, permissionsByUser map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(segments) < 2 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		username := segments[len(segments)-2]

		perm, ok := permissionsByUser[username]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"permission": perm})
	}))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newTestAuthorizerWithServer(server *httptest.Server) *Authorizer {
	return &Authorizer{
		githubToken: "fake-token",
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				req.URL.Scheme = "http"
				req.URL.Host = server.Listener.Addr().String()
				req.Host = server.Listener.Addr().String()
				return http.DefaultTransport.RoundTrip(req)
			}),
		},
	}
}

func TestGetCollaboratorPermission(t *testing.T) {
	tests := []struct {
		name           string
		username       string
		serverResponse map[string]string
		wantLevel      PermissionLevel
		wantCanWrite   bool
	}{
		{
			name:           "admin can write",
			username:       "alice",
			serverResponse: map[string]string{"alice": "admin"},
			wantLevel:      PermissionAdmin,
			wantCanWrite:   true,
		},
		{
			name:           "write permission can write",
			username:       "bob",
			serverResponse: map[string]string{"bob": "write"},
			wantLevel:      PermissionWrite,
			wantCanWrite:   true,
		},
		{
			name:           "read permission cannot write",
			username:       "carol",
			serverResponse: map[string]string{"carol": "read"},
			wantLevel:      PermissionRead,
			wantCanWrite:   false,
		},
		{
			name:           "none permission cannot write",
			username:       "dave",
			serverResponse: map[string]string{"dave": "none"},
			wantLevel:      PermissionNone,
			wantCanWrite:   false,
		},
		{
			name:           "not a collaborator at all (404) treated as none",
			username:       "eve",
			serverResponse: map[string]string{},
			wantLevel:      PermissionNone,
			wantCanWrite:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := fakeGitHubServer(t, tt.serverResponse)
			defer server.Close()

			a := newTestAuthorizerWithServer(server)

			level, err := a.getCollaboratorPermission(context.Background(), "owner", "repo", tt.username)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if level != tt.wantLevel {
				t.Errorf("level = %v, want %v", level, tt.wantLevel)
			}
			if level.canWrite() != tt.wantCanWrite {
				t.Errorf("canWrite() = %v, want %v", level.canWrite(), tt.wantCanWrite)
			}
		})
	}
}