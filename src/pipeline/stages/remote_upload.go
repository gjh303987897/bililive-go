package stages

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bililive-go/bililive-go/src/configs"
	"github.com/bililive-go/bililive-go/src/pipeline"
	"github.com/bililive-go/bililive-go/src/pkg/remoteupload"
)

const (
	// OptionRemotePathTmpl 覆盖全局远程上传路径模板
	OptionRemotePathTmpl = "path_tmpl"
)

// RemoteUploadStage 将转码产物上传到 WebDAV / S3。
type RemoteUploadStage struct {
	config    pipeline.StageConfig
	pathTmpl  string
	deleteAll bool
	commands  []string
	logs      string
	logger    pipelineLogger
}

type pipelineLogger interface {
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
}

// NewRemoteUploadStage 创建远程上传阶段。
func NewRemoteUploadStage(config pipeline.StageConfig) (pipeline.Stage, error) {
	return &RemoteUploadStage{
		config:    config,
		pathTmpl:  config.GetStringOption(OptionRemotePathTmpl, ""),
		deleteAll: config.GetBoolOption(pipeline.OptionDeleteSource, false),
	}, nil
}

func (s *RemoteUploadStage) Name() string {
	return pipeline.StageNameRemoteUpload
}

func (s *RemoteUploadStage) GetCommands() []string {
	return s.commands
}

func (s *RemoteUploadStage) GetLogs() string {
	return s.logs
}

func (s *RemoteUploadStage) Execute(ctx *pipeline.PipelineContext, input []pipeline.FileInfo) ([]pipeline.FileInfo, error) {
	if len(input) == 0 {
		return input, nil
	}
	s.logger = ctx.Logger
	cfg := configs.GetCurrentConfig()
	if cfg == nil || !cfg.RemoteUpload.Enable {
		s.logs = "远程上传未启用，跳过"
		return input, nil
	}

	provider, err := remoteupload.New(cfg.RemoteUpload)
	if err != nil {
		s.logs += fmt.Sprintf("初始化上传客户端失败: %s\n", err.Error())
		return input, err
	}

	pathTmpl := s.pathTmpl
	if pathTmpl == "" {
		pathTmpl = cfg.RemoteUpload.PathTmpl
	}

	output := make([]pipeline.FileInfo, 0, len(input))
	var failures []string
	for _, file := range input {
		output = append(output, file)
		if file.Type != pipeline.FileTypeVideo || file.Deletable {
			continue
		}
		if info, err := os.Stat(file.Path); err != nil || info.IsDir() {
			continue
		}

		remotePath, err := renderRemotePath(pathTmpl, ctx, file)
		if err != nil {
			msg := fmt.Sprintf("渲染远程路径失败: %s - %s", file.Path, err.Error())
			s.logs += msg + "\n"
			failures = append(failures, msg)
			continue
		}
		if remotePath == "" {
			msg := fmt.Sprintf("远程路径为空: %s", file.Path)
			s.logs += msg + "\n"
			failures = append(failures, msg)
			continue
		}

		displayPath := remoteupload.JoinRemotePath(cfg.RemoteUpload.BasePath(), remotePath)
		s.commands = append(s.commands, fmt.Sprintf("upload %s -> %s", file.Path, displayPath))
		s.logger.Infof("远程上传: %s -> %s", file.Path, displayPath)
		if err := remoteupload.UploadWithRetry(ctx.Ctx, provider, file.Path, remotePath, cfg.RemoteUpload.RetryLimit); err != nil {
			msg := fmt.Sprintf("远程上传失败: %s - %s", file.Path, err.Error())
			s.logs += msg + "\n"
			failures = append(failures, msg)
			continue
		}
		file.Metadata = ensureMetadata(file.Metadata)
		file.Metadata["remote_path"] = displayPath
		output[len(output)-1] = file
		if s.deleteAll {
			file.Deletable = true
			output[len(output)-1] = file
		}
		s.logs += fmt.Sprintf("远程上传完成: %s -> %s\n", filepath.Base(file.Path), displayPath)
	}

	if len(failures) > 0 {
		return output, fmt.Errorf("远程上传失败 %d 个文件: %s", len(failures), strings.Join(failures, "; "))
	}
	return output, nil
}

func ensureMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return make(map[string]any)
	}
	return metadata
}

func renderRemotePath(tmpl string, ctx *pipeline.PipelineContext, file pipeline.FileInfo) (string, error) {
	data := pipeline.UploadPathData{
		Platform: ctx.RecordInfo.Platform,
		HostName: ctx.RecordInfo.HostName,
		RoomName: ctx.RecordInfo.RoomName,
		FileName: filepath.Base(file.Path),
		Ext:      filepath.Ext(file.Path),
	}
	return pipeline.RenderUploadPath(tmpl, data, func() time.Time {
		if !ctx.RecordInfo.StartTime.IsZero() {
			return ctx.RecordInfo.StartTime
		}
		return time.Now()
	})
}
