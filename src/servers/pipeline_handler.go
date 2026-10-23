package servers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/bililive-go/bililive-go/src/configs"
	"github.com/bililive-go/bililive-go/src/pipeline"
	"github.com/bililive-go/bililive-go/src/types"
)

// RegisterPipelineHandlers 注册 Pipeline 任务管理相关的 HTTP 处理器
// 注意：r 已经是 /api 前缀的子路由器
func RegisterPipelineHandlers(r *mux.Router, pm *pipeline.Manager) {
	if pm == nil {
		return
	}

	// 获取 Pipeline 任务列表
	r.HandleFunc("/pipeline/tasks", makePipelineListTasksHandler(pm)).Methods("GET")

	// 获取队列统计
	r.HandleFunc("/pipeline/tasks/stats", makePipelineGetStatsHandler(pm)).Methods("GET")

	// 清除已完成的任务
	r.HandleFunc("/pipeline/tasks/clear-completed", makePipelineClearCompletedHandler(pm)).Methods("POST")

	// 获取单个任务
	r.HandleFunc("/pipeline/tasks/{id}", makePipelineGetTaskHandler(pm)).Methods("GET")

	// 取消任务
	r.HandleFunc("/pipeline/tasks/{id}/cancel", makePipelineCancelTaskHandler(pm)).Methods("POST")

	// 重试任务
	r.HandleFunc("/pipeline/tasks/{id}/retry", makePipelineRetryTaskHandler(pm)).Methods("POST")

	// 删除任务
	r.HandleFunc("/pipeline/tasks/{id}", makePipelineDeleteTaskHandler(pm)).Methods("DELETE")

	// 批量烧录弹幕字幕
	r.HandleFunc("/pipeline/batch-burn", makeBatchBurnHandler(pm)).Methods("POST")

	// 队列视图
	r.HandleFunc("/pipeline/queues", makePipelineQueuesHandler(pm)).Methods("GET")
	r.HandleFunc("/pipeline/queues/transcode/clear-failed", makePipelineClearFailedHandler(pm)).Methods("POST")
}

// makePipelineListTasksHandler 列出 Pipeline 任务
func makePipelineListTasksHandler(pm *pipeline.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := pipeline.TaskFilter{}

		// 解析查询参数
		if status := r.URL.Query().Get("status"); status != "" {
			s := pipeline.PipelineStatus(status)
			filter.Status = &s
		}
		if liveID := r.URL.Query().Get("live_id"); liveID != "" {
			filter.LiveID = &liveID
		}
		if kind := r.URL.Query().Get("kind"); kind != "" {
			k := pipeline.TaskKind(kind)
			filter.Kind = &k
		}
		if limit := r.URL.Query().Get("limit"); limit != "" {
			if l, err := strconv.Atoi(limit); err == nil {
				filter.Limit = l
			}
		}
		if offset := r.URL.Query().Get("offset"); offset != "" {
			if o, err := strconv.Atoi(offset); err == nil {
				filter.Offset = o
			}
		}

		tasks, err := pm.ListTasks(filter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 确保返回空数组而不是 null
		if tasks == nil {
			tasks = []*pipeline.PipelineTask{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	}
}

// makePipelineGetStatsHandler 获取队列统计
func makePipelineGetStatsHandler(pm *pipeline.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := pm.GetStats()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

// makePipelineClearCompletedHandler 清除已完成的任务
func makePipelineClearCompletedHandler(pm *pipeline.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		count, err := pm.ClearCompletedTasks()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"deleted": count,
		})
	}
}

// makePipelineGetTaskHandler 获取单个任务
func makePipelineGetTaskHandler(pm *pipeline.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			http.Error(w, "invalid task id", http.StatusBadRequest)
			return
		}

		task, err := pm.GetTask(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if task == nil {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
	}
}

// makePipelineCancelTaskHandler 取消任务
func makePipelineCancelTaskHandler(pm *pipeline.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			http.Error(w, "invalid task id", http.StatusBadRequest)
			return
		}

		if err := pm.CancelTask(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
	}
}

// makePipelineRetryTaskHandler 重试任务
// 支持可选的 JSON body: {"stage_name": "cloud_upload"}
// 无 body 或 stage_name 为空时全量重试
func makePipelineRetryTaskHandler(pm *pipeline.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			http.Error(w, "invalid task id", http.StatusBadRequest)
			return
		}

		// 解析可选的 JSON body
		var reqBody struct {
			StageName string `json:"stage_name"`
		}
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil && err != io.EOF {
				http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		if reqBody.StageName != "" {
			// 从指定阶段重试
			if err := pm.RetryTaskFromStage(id, reqBody.StageName); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			// 全量重试（向后兼容）
			if err := pm.RetryTask(id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "retried"})
	}
}

// makePipelineDeleteTaskHandler 删除任务
func makePipelineDeleteTaskHandler(pm *pipeline.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			http.Error(w, "invalid task id", http.StatusBadRequest)
			return
		}

		if err := pm.DeleteTask(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	}
}

// isVideoFile 判断文件扩展名是否为常见视频格式
func isVideoFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".mp4", ".flv", ".ts", ".mkv", ".avi", ".mov", ".wmv", ".webm":
		return true
	}
	return false
}

// makeBatchBurnHandler 批量烧录弹幕字幕
func makeBatchBurnHandler(pm *pipeline.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Paths []string `json:"paths"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if len(req.Paths) == 0 {
			http.Error(w, "paths is required", http.StatusBadRequest)
			return
		}

		config := configs.GetCurrentConfig()
		if config == nil {
			http.Error(w, "config not available", http.StatusInternalServerError)
			return
		}

		// 读取当前烧录配置
		legacy := &config.OnRecordFinished
		codec := legacy.BurnSubtitlesCodec
		if codec == "" {
			codec = "libx264"
		}
		crf := legacy.BurnSubtitlesCrf
		if crf == "" {
			crf = "18"
		}
		preset := legacy.BurnSubtitlesPreset
		if preset == "" {
			preset = "medium"
		}

		// 构建仅含 burn_subtitles 阶段的 PipelineConfig
		burnConfig := &pipeline.PipelineConfig{
			Stages: []pipeline.StageConfig{
				{
					Name: pipeline.StageNameBurnSubtitles,
					Options: map[string]any{
						pipeline.OptionCodec:            codec,
						pipeline.OptionCrf:              crf,
						pipeline.OptionPreset:           preset,
						pipeline.OptionBurnDeleteAss:    legacy.BurnDeleteAss,
						pipeline.OptionBurnDeleteSource: legacy.BurnDeleteSource,
					},
				},
			},
		}

		outputPath := config.OutPutPath

		type batchResult struct {
			Enqueued int      `json:"enqueued"`
			Skipped  []string `json:"skipped"`
			TaskIDs  []int64  `json:"task_ids"`
		}
		result := batchResult{
			Skipped: []string{},
		}

		for _, p := range req.Paths {
			// 校验路径安全性，防止路径遍历攻击
			absPath, err := getSafePath(outputPath, p)
			if err != nil {
				result.Skipped = append(result.Skipped, filepath.Base(p)+" - 路径越权")
				continue
			}

			// 验证文件存在
			if _, err := os.Stat(absPath); err != nil {
				result.Skipped = append(result.Skipped, filepath.Base(p)+" - 文件不存在或无法访问")
				continue
			}

			// 检查是否为视频文件
			if !isVideoFile(absPath) {
				result.Skipped = append(result.Skipped, filepath.Base(p)+" - 非视频文件")
				continue
			}

			// 检查同名 ASS 文件是否存在
			baseName := absPath[:len(absPath)-len(filepath.Ext(absPath))]
			assPath := baseName + ".ass"
			if _, err := os.Stat(assPath); os.IsNotExist(err) {
				result.Skipped = append(result.Skipped, filepath.Base(p)+" - 无 ASS 字幕文件")
				continue
			}

			// 计算相对路径用于 RecordInfo（显示用）
			relPath := p
			if filepath.IsAbs(p) {
				if rel, err := filepath.Rel(outputPath, p); err == nil {
					relPath = rel
				}
			}

			// 创建任务
			files := []pipeline.FileInfo{pipeline.NewVideoFileInfo(absPath)}
			recordInfo := pipeline.RecordInfo{
				LiveID:    types.LiveID("batch-burn"),
				Platform:  "手动批量烧录",
				HostName:  relPath,
				RoomName:  "",
				StartTime: time.Now(),
			}
			task := pipeline.NewPipelineTask(recordInfo, burnConfig, files)

			if err := pm.EnqueueTask(task, nil); err != nil {
				result.Skipped = append(result.Skipped, filepath.Base(p)+" - 入队失败: "+err.Error())
				continue
			}

			result.Enqueued++
			result.TaskIDs = append(result.TaskIDs, task.ID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

type pipelineQueueTask struct {
	ID        int64                   `json:"id"`
	Status    pipeline.PipelineStatus `json:"status"`
	Files     []pipeline.FileInfo     `json:"files"`
	Record    pipeline.RecordInfo     `json:"record_info"`
	Error     string                  `json:"error_message,omitempty"`
	Progress  int                     `json:"progress"`
	CreatedAt time.Time               `json:"created_at"`
	CanRetry  bool                    `json:"can_retry"`
}

func toQueueTask(task *pipeline.PipelineTask) pipelineQueueTask {
	return pipelineQueueTask{
		ID:        task.ID,
		Status:    task.Status,
		Files:     task.CurrentFiles,
		Record:    task.RecordInfo,
		Error:     task.ErrorMessage,
		Progress:  task.Progress,
		CreatedAt: task.CreatedAt,
		CanRetry:  task.CanRetry,
	}
}

func listQueue(pm *pipeline.Manager, kind pipeline.TaskKind, statuses ...pipeline.PipelineStatus) ([]pipelineQueueTask, error) {
	result := make([]pipelineQueueTask, 0, 16)
	for _, status := range statuses {
		s := status
		tasks, err := pm.ListTasks(pipeline.TaskFilter{Status: &s, Kind: &kind, Limit: 100})
		if err != nil {
			return nil, err
		}
		for _, task := range tasks {
			result = append(result, toQueueTask(task))
		}
	}
	return result, nil
}

// makePipelineQueuesHandler 返回等待转码、转码失败和上传队列。
func makePipelineQueuesHandler(pm *pipeline.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		transcodeWaiting, err := listQueue(pm, pipeline.TaskKindTranscode, pipeline.PipelineStatusPending, pipeline.PipelineStatusRunning)
		if err != nil {
			writeJsonWithStatusCode(w, http.StatusInternalServerError, commonResp{ErrNo: 1, ErrMsg: err.Error()})
			return
		}
		transcodeFailed, err := listQueue(pm, pipeline.TaskKindTranscode, pipeline.PipelineStatusFailed)
		if err != nil {
			writeJsonWithStatusCode(w, http.StatusInternalServerError, commonResp{ErrNo: 1, ErrMsg: err.Error()})
			return
		}
		uploadPending, err := listQueue(pm, pipeline.TaskKindUpload, pipeline.PipelineStatusPending, pipeline.PipelineStatusRunning)
		if err != nil {
			writeJsonWithStatusCode(w, http.StatusInternalServerError, commonResp{ErrNo: 1, ErrMsg: err.Error()})
			return
		}
		uploadFailed, err := listQueue(pm, pipeline.TaskKindUpload, pipeline.PipelineStatusFailed)
		if err != nil {
			writeJsonWithStatusCode(w, http.StatusInternalServerError, commonResp{ErrNo: 1, ErrMsg: err.Error()})
			return
		}
		uploadCompleted, err := listQueue(pm, pipeline.TaskKindUpload, pipeline.PipelineStatusCompleted)
		if err != nil {
			writeJsonWithStatusCode(w, http.StatusInternalServerError, commonResp{ErrNo: 1, ErrMsg: err.Error()})
			return
		}

		cfg := configs.GetCurrentConfig()
		uploadEnabled := cfg != nil && cfg.RemoteUpload.Enable
		writeJSON(w, commonResp{Data: map[string]any{
			"transcode_waiting": transcodeWaiting,
			"transcode_failed":  transcodeFailed,
			"upload_pending":    uploadPending,
			"upload_failed":     uploadFailed,
			"upload_completed":  uploadCompleted,
			"upload_enabled":    uploadEnabled,
		}})
	}
}

// makePipelineClearFailedHandler 清空转码失败队列（保留本地文件）。
func makePipelineClearFailedHandler(pm *pipeline.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := pipeline.PipelineStatusFailed
		kind := pipeline.TaskKindTranscode
		tasks, err := pm.ListTasks(pipeline.TaskFilter{Status: &status, Kind: &kind, Limit: 1000})
		if err != nil {
			writeJsonWithStatusCode(w, http.StatusInternalServerError, commonResp{ErrNo: 1, ErrMsg: err.Error()})
			return
		}
		deleted := 0
		for _, task := range tasks {
			if err := pm.DeleteTask(task.ID); err != nil {
				continue
			}
			deleted++
		}
		writeJSON(w, commonResp{Data: map[string]int{"deleted": deleted}})
	}
}
