package servers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bililive-go/bililive-go/src/pipeline"
)

func newQueueTestManager(t *testing.T) *pipeline.Manager {
	t.Helper()
	store, err := pipeline.NewSQLiteStore(t.TempDir() + "/pipeline.db")
	require.NoError(t, err)
	manager := pipeline.NewManager(context.Background(), store, nil, nil)
	t.Cleanup(func() { manager.Close(context.Background()) })
	return manager
}

func TestPipelineQueuesSeparateTranscodeAndUpload(t *testing.T) {
	manager := newQueueTestManager(t)
	files := []pipeline.FileInfo{{Path: "a.mp4", Type: pipeline.FileTypeVideo}}
	record := pipeline.RecordInfo{LiveID: "live-1", HostName: "主播"}

	waiting := pipeline.NewPipelineTask(record, &pipeline.PipelineConfig{
		Stages: []pipeline.StageConfig{{Name: pipeline.StageNameConvertMp4}},
	}, files)
	require.NoError(t, manager.CreateTask(waiting))

	failed := pipeline.NewPipelineTask(record, &pipeline.PipelineConfig{
		Stages: []pipeline.StageConfig{{Name: pipeline.StageNameConvertMp4}},
	}, files)
	failed.MarkFailed(errors.New("转码失败"))
	require.NoError(t, manager.CreateTask(failed))

	upload := pipeline.NewPipelineTask(record, &pipeline.PipelineConfig{
		Stages: []pipeline.StageConfig{{Name: pipeline.StageNameRemoteUpload}},
	}, files)
	require.NoError(t, manager.CreateTask(upload))

	rec := httptest.NewRecorder()
	makePipelineQueuesHandler(manager)(rec, httptest.NewRequest("GET", "/api/pipeline/queues", nil))
	require.Equal(t, 200, rec.Code)

	var payload struct {
		Data struct {
			TranscodeWaiting []pipelineQueueTask `json:"transcode_waiting"`
			TranscodeFailed  []pipelineQueueTask `json:"transcode_failed"`
			UploadPending    []pipelineQueueTask `json:"upload_pending"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))

	require.Len(t, payload.Data.TranscodeWaiting, 1)
	assert.Equal(t, waiting.ID, payload.Data.TranscodeWaiting[0].ID)
	require.Len(t, payload.Data.TranscodeFailed, 1)
	assert.Equal(t, failed.ID, payload.Data.TranscodeFailed[0].ID)
	require.Len(t, payload.Data.UploadPending, 1)
	assert.Equal(t, upload.ID, payload.Data.UploadPending[0].ID)
}

func TestPipelineClearFailedRemovesOnlyFailed(t *testing.T) {
	manager := newQueueTestManager(t)
	record := pipeline.RecordInfo{LiveID: "live-1"}
	config := &pipeline.PipelineConfig{Stages: []pipeline.StageConfig{{Name: pipeline.StageNameConvertMp4}}}

	pending := pipeline.NewPipelineTask(record, config, nil)
	require.NoError(t, manager.CreateTask(pending))
	failed := pipeline.NewPipelineTask(record, config, nil)
	failed.MarkFailed(errors.New("boom"))
	require.NoError(t, manager.CreateTask(failed))

	rec := httptest.NewRecorder()
	makePipelineClearFailedHandler(manager)(rec, httptest.NewRequest("POST", "/api/pipeline/queues/transcode/clear-failed", nil))
	require.Equal(t, 200, rec.Code)

	tasks, err := manager.ListTasks(pipeline.TaskFilter{})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, pending.ID, tasks[0].ID)
}
