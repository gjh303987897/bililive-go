package remoteupload

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bililive-go/bililive-go/src/configs"
)

func TestJoinRemotePath(t *testing.T) {
	assert.Equal(t, "base/a/b.mp4", JoinRemotePath("/base/", "/a/b.mp4"))
	assert.Equal(t, "a/b.mp4", JoinRemotePath("", "a/b.mp4"))
	assert.Equal(t, "base/a/a", JoinRemotePath("base", `a\b\..\a`))
}

func TestWebDAVUploadCreatesDirectoriesAndUploads(t *testing.T) {
	var mu sync.Mutex
	var created []string
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		user, pass, ok := r.BasicAuth()
		if !ok || user != "user" || pass != "pass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case "MKCOL":
			created = append(created, r.URL.Path)
			w.WriteHeader(http.StatusCreated)
		case http.MethodPut:
			data, _ := io.ReadAll(r.Body)
			body = string(data)
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	local := filepath.Join(dir, "video.mp4")
	require.NoError(t, os.WriteFile(local, []byte("payload"), 0644))

	provider, err := newWebDAV(configs.WebDAVUpload{URL: server.URL, BasePath: "/base", Username: "user", Password: "pass"})
	require.NoError(t, err)
	require.NoError(t, provider.Upload(context.Background(), local, "a/b/video.mp4"))

	assert.Equal(t, "payload", body)
	assert.Equal(t, []string{"/base", "/base/a", "/base/a/b"}, created)
}

func TestWebDAVUploadRejectsBadURL(t *testing.T) {
	_, err := newWebDAV(configs.WebDAVUpload{URL: "not-a-url"})
	assert.Error(t, err)
}

func TestUploadWithRetryStopsAfterSuccess(t *testing.T) {
	attempts := 0
	provider := providerFunc(func(context.Context, string, string) error {
		attempts++
		if attempts < 3 {
			return assert.AnError
		}
		return nil
	})
	require.NoError(t, UploadWithRetry(context.Background(), provider, "local", "remote", 5))
	assert.Equal(t, 3, attempts)
}

type providerFunc func(context.Context, string, string) error

func (f providerFunc) Upload(ctx context.Context, local, remote string) error {
	return f(ctx, local, remote)
}
