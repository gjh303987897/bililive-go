package stages

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bililive-go/bililive-go/src/configs"
	"github.com/bililive-go/bililive-go/src/pipeline"
	"github.com/bililive-go/bililive-go/src/pkg/livelogger"
)

func TestRemoteUploadStageUploadsVideo(t *testing.T) {
	var uploaded []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusCreated)
			return
		}
		uploaded, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	previous := configs.GetCurrentConfig()
	defer configs.SetCurrentConfig(previous)
	config := configs.NewConfig()
	config.RemoteUpload.Enable = true
	config.RemoteUpload.Provider = "webdav"
	config.RemoteUpload.WebDAV.URL = server.URL
	config.RemoteUpload.WebDAV.BasePath = "/bgo"
	config.RemoteUpload.PathTmpl = "/{{ .Platform }}/{{ .FileName }}"
	configs.SetCurrentConfig(config)

	dir := t.TempDir()
	path := filepath.Join(dir, "video.mp4")
	require.NoError(t, os.WriteFile(path, []byte("video-bytes"), 0644))

	stage, err := NewRemoteUploadStage(pipeline.StageConfig{})
	require.NoError(t, err)
	ctx := &pipeline.PipelineContext{
		Ctx:        context.Background(),
		Logger:     livelogger.New(10, nil),
		RecordInfo: pipeline.RecordInfo{Platform: "示例"},
	}
	output, err := stage.Execute(ctx, []pipeline.FileInfo{{Path: path, Type: pipeline.FileTypeVideo}})
	require.NoError(t, err, stage.(*RemoteUploadStage).GetLogs())
	require.Len(t, output, 1)
	assert.Equal(t, "video-bytes", string(uploaded))
	assert.Equal(t, "bgo/示例/video.mp4", output[0].Metadata["remote_path"])
}

func TestRemoteUploadStageSkipsWhenDisabled(t *testing.T) {
	previous := configs.GetCurrentConfig()
	defer configs.SetCurrentConfig(previous)
	config := configs.NewConfig()
	config.RemoteUpload.Enable = false
	configs.SetCurrentConfig(config)

	stage, err := NewRemoteUploadStage(pipeline.StageConfig{})
	require.NoError(t, err)
	input := []pipeline.FileInfo{{Path: "x.mp4", Type: pipeline.FileTypeVideo}}
	output, err := stage.Execute(&pipeline.PipelineContext{Ctx: context.Background(), Logger: livelogger.New(10, nil)}, input)
	require.NoError(t, err)
	assert.Equal(t, input, output)
}

func TestRemoteUploadStageReportsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	previous := configs.GetCurrentConfig()
	defer configs.SetCurrentConfig(previous)
	config := configs.NewConfig()
	config.RemoteUpload.Enable = true
	config.RemoteUpload.Provider = "webdav"
	config.RemoteUpload.WebDAV.URL = server.URL
	config.RemoteUpload.WebDAV.BasePath = "/bgo"
	config.RemoteUpload.RetryLimit = 1
	configs.SetCurrentConfig(config)

	dir := t.TempDir()
	path := filepath.Join(dir, "video.mp4")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))

	stage, err := NewRemoteUploadStage(pipeline.StageConfig{})
	require.NoError(t, err)
	ctx := &pipeline.PipelineContext{Ctx: context.Background(), Logger: livelogger.New(10, nil)}
	output, err := stage.Execute(ctx, []pipeline.FileInfo{{Path: path, Type: pipeline.FileTypeVideo}})
	require.Error(t, err)
	require.Len(t, output, 1)
	assert.Contains(t, stage.(*RemoteUploadStage).GetLogs(), "远程上传失败")
}
