package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bililive-go/bililive-go/src/configs"
)

func TestEnqueueRemoteUploadCreatesUploadTask(t *testing.T) {
	previous := configs.GetCurrentConfig()
	defer configs.SetCurrentConfig(previous)
	config := configs.NewConfig()
	config.RemoteUpload.Enable = true
	configs.SetCurrentConfig(config)

	store, err := NewSQLiteStore(t.TempDir() + "/pipeline.db")
	require.NoError(t, err)
	manager := NewManager(context.Background(), store, nil, nil)
	t.Cleanup(func() { manager.Close(context.Background()) })

	dir := t.TempDir()
	video := filepath.Join(dir, "video.mp4")
	require.NoError(t, os.WriteFile(video, []byte("x"), 0644))

	task := NewPipelineTask(RecordInfo{LiveID: "live"}, &PipelineConfig{
		Stages: []StageConfig{{Name: StageNameConvertMp4}},
	}, []FileInfo{{Path: filepath.Join(dir, "source.flv"), Type: FileTypeVideo}})
	require.NoError(t, manager.EnqueueTask(task, nil))

	manager.enqueueRemoteUpload(task, []StageResult{{
		OutputFiles: []FileInfo{
			{Path: video, Type: FileTypeVideo},
			{Path: filepath.Join(dir, "source.flv"), Type: FileTypeVideo, Deletable: true},
		},
	}})

	tasks, err := manager.ListTasks(TaskFilter{})
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	var upload *PipelineTask
	for _, item := range tasks {
		if item.ID != task.ID {
			upload = item
		}
	}
	require.NotNil(t, upload)
	assert.Equal(t, TaskKindUpload, upload.Kind())
	require.Len(t, upload.InitialFiles, 1)
	assert.Equal(t, video, upload.InitialFiles[0].Path)
}

func TestEnqueueRemoteUploadSkipsDisabled(t *testing.T) {
	previous := configs.GetCurrentConfig()
	defer configs.SetCurrentConfig(previous)
	config := configs.NewConfig()
	config.RemoteUpload.Enable = false
	configs.SetCurrentConfig(config)

	store, err := NewSQLiteStore(t.TempDir() + "/pipeline.db")
	require.NoError(t, err)
	manager := NewManager(context.Background(), store, nil, nil)
	t.Cleanup(func() { manager.Close(context.Background()) })

	task := NewPipelineTask(RecordInfo{}, &PipelineConfig{Stages: []StageConfig{{Name: StageNameConvertMp4}}}, nil)
	require.NoError(t, manager.EnqueueTask(task, nil))
	manager.enqueueRemoteUpload(task, []StageResult{{OutputFiles: []FileInfo{{Path: "x.mp4", Type: FileTypeVideo}}}})

	tasks, err := manager.ListTasks(TaskFilter{})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}
