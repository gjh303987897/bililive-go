// Package remoteupload 提供转码产物直连远程存储的上传实现。
package remoteupload

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/bililive-go/bililive-go/src/configs"
)

// Provider 远程上传接口。
type Provider interface {
	// Upload 将本地文件上传到 remotePath。remotePath 为相对路径（已按模板渲染）。
	Upload(ctx context.Context, localPath, remotePath string) error
}

// New 根据配置创建上传实现。
func New(cfg configs.RemoteUpload) (Provider, error) {
	switch cfg.Provider {
	case "webdav":
		return newWebDAV(cfg.WebDAV)
	case "s3":
		return newS3(cfg.S3)
	default:
		return nil, fmt.Errorf("未知的上传方式: %q", cfg.Provider)
	}
}

// JoinRemotePath 拼接远程根目录与相对路径，保留正斜杠。
func JoinRemotePath(basePath, remotePath string) string {
	cleaned := path.Join("/", strings.ReplaceAll(basePath, "\\", "/"), strings.ReplaceAll(remotePath, "\\", "/"))
	return strings.TrimPrefix(cleaned, "/")
}

type webDAVProvider struct {
	base     *url.URL
	client   *http.Client
	username string
	password string
}

func newWebDAV(cfg configs.WebDAVUpload) (Provider, error) {
	parsed, err := url.Parse(strings.TrimSpace(cfg.URL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("WebDAV 地址无效: %q", cfg.URL)
	}
	parsed.Path = path.Join(parsed.Path, strings.ReplaceAll(cfg.BasePath, "\\", "/"))
	return &webDAVProvider{
		base:     parsed,
		client:   &http.Client{Timeout: 0},
		username: cfg.Username,
		password: cfg.Password,
	}, nil
}

func (p *webDAVProvider) Upload(ctx context.Context, localPath, remotePath string) error {
	target := *p.base
	target.Path = path.Join(p.base.Path, strings.ReplaceAll(remotePath, "\\", "/"))

	if err := p.mkdirAll(ctx, path.Dir(target.Path), target); err != nil {
		return err
	}

	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("读取本地文件信息失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, target.String(), file)
	if err != nil {
		return err
	}
	req.ContentLength = info.Size()
	if contentType := mime.TypeByExtension(filepath.Ext(localPath)); contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if p.username != "" {
		req.SetBasicAuth(p.username, p.password)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("WebDAV 上传失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("WebDAV 返回状态 %d", resp.StatusCode)
	}
	return nil
}

// mkdirAll 逐级创建远程目录；已存在的目录返回 405 时忽略。
func (p *webDAVProvider) mkdirAll(ctx context.Context, dir string, target url.URL) error {
	dir = path.Clean(dir)
	if dir == "/" || dir == "." {
		return nil
	}
	var current string
	for _, segment := range strings.Split(strings.TrimPrefix(dir, "/"), "/") {
		if segment == "" {
			continue
		}
		current = path.Join(current, segment)
		reqURL := target
		reqURL.Path = "/" + current
		req, err := http.NewRequestWithContext(ctx, "MKCOL", reqURL.String(), nil)
		if err != nil {
			return err
		}
		if p.username != "" {
			req.SetBasicAuth(p.username, p.password)
		}
		resp, err := p.client.Do(req)
		if err != nil {
			return fmt.Errorf("创建 WebDAV 目录失败: %w", err)
		}
		resp.Body.Close()
		if resp.StatusCode >= 300 && resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusConflict {
			return fmt.Errorf("创建 WebDAV 目录 %s 失败: 状态 %d", current, resp.StatusCode)
		}
	}
	return nil
}

type s3Provider struct {
	client   *minio.Client
	bucket   string
	basePath string
}

func newS3(cfg configs.S3Upload) (Provider, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	// minio 需要不带协议头的 endpoint
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	secure := cfg.UseSSL
	if strings.HasPrefix(cfg.Endpoint, "http://") {
		secure = false
	} else if strings.HasPrefix(cfg.Endpoint, "https://") {
		secure = true
	}

	options := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: secure,
		Region: cfg.Region,
	}
	if cfg.PathStyle {
		options.BucketLookup = minio.BucketLookupPath
	}
	client, err := minio.New(endpoint, options)
	if err != nil {
		return nil, fmt.Errorf("创建 S3 客户端失败: %w", err)
	}
	return &s3Provider{client: client, bucket: cfg.Bucket, basePath: cfg.BasePath}, nil
}

func (p *s3Provider) Upload(ctx context.Context, localPath, remotePath string) error {
	object := JoinRemotePath(p.basePath, remotePath)
	options := minio.PutObjectOptions{}
	if contentType := mime.TypeByExtension(filepath.Ext(localPath)); contentType != "" {
		options.ContentType = contentType
	}
	_, err := p.client.FPutObject(ctx, p.bucket, object, localPath, options)
	if err != nil {
		return fmt.Errorf("S3 上传失败: %w", err)
	}
	return nil
}

// UploadWithRetry 按次数重试上传；每次重试前遵守 ctx 取消。
func UploadWithRetry(ctx context.Context, provider Provider, localPath, remotePath string, retryLimit int) error {
	var err error
	for attempt := 0; attempt <= retryLimit; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}
		if err = provider.Upload(ctx, localPath, remotePath); err == nil {
			return nil
		}
	}
	return err
}
