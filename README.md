# Bililive-go

[![Publish Container Image](https://github.com/bililive-go/bililive-go/actions/workflows/ghcr.yaml/badge.svg?branch=master)](https://github.com/bililive-go/bililive-go/actions/workflows/ghcr.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/bililive-go/bililive-go)](https://goreportcard.com/report/github.com/bililive-go/bililive-go)
[![Github release](https://img.shields.io/github/release/bililive-go/bililive-go.svg)](https://github.com/bililive-go/bililive-go/releases/latest)
[![Docker Pulls](https://img.shields.io/docker/pulls/chigusa/bililive-go.svg)](https://hub.docker.com/r/chigusa/bililive-go/)
[![Bilibili](https://img.shields.io/badge/%E5%93%94%E5%93%A9%E5%93%94%E5%93%A9-%E6%9C%AA%E6%9D%A5%E7%A7%91%E6%8A%80%E7%8E%8B%E8%80%81%E8%8F%8A%E5%BD%95%E6%92%AD%E7%BB%84-ebb8d0.svg)](https://space.bilibili.com/18578203/)

Bililive-go 是一个支持多种直播平台的直播录制工具，提供网页管理、实时监控、后处理管道、失败重试与远程归档。

### 第三方生态

> 安全提示：Bililive-go 未审计任何第三方代码的安全性、合规性和可用性。是否安装或使用来自第三方的脚本、插件、扩展等内容，请自行判断并承担相应风险。
>
> 社区维护的第三方生态项目列表见：[docs/third-party.md](docs/third-party.md)

![image](docs/screenshot.webp)

## 功能

- **多平台录制**：支持下列平台，自动检测开播、分段、断线重连
- **内置监控面板**：直播/录制状态、下载速率、开机时长排行、每日开播时长、单房间历史、内存与磁盘 IO，不依赖 Prometheus / Grafana
- **后处理队列**：等待转码、转码失败（源文件保留）、上传队列可视化，支持重试、取消与清空
- **远程归档**：转码完成后可自动上传到 WebDAV 或 S3 兼容存储，带重试与路径模板
- **通知**：Telegram、ntfy、邮件、Bark、WxPusher 等开播/下播/异常通知
- **界面**：中文 / English 切换，明亮 / 暗黑主题，选择会保存在浏览器
- **其他**：浏览器内播放器、弹幕录制与烧录、云上传（OpenList）、自动更新、Docker 多架构镜像

## 支持网站

<table>
    <tr align="center">
        <th>站点</th>
        <th>url</th>
        <th>支持情况</th>
        <th>cookie</th>
    </tr>
    <tr align="center">
        <td>Acfun直播</td>
        <td>live.acfun.cn</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>哔哩哔哩直播</td>
        <td>live.bilibili.com</td>
        <td>✅ 支持</td>
        <td>✅ 支持</td>
    </tr>
    <tr align="center">
        <td>战旗直播</td>
        <td>www.zhanqi.tv</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>斗鱼直播</td>
        <td>www.douyu.com</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>虎牙直播</td>
        <td>www.huya.com</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>CC直播</td>
        <td>cc.163.com</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>一直播</td>
        <td>www.yizhibo.com</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>OPENREC</td>
        <td>www.openrec.tv</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>企鹅电竞</td>
        <td>egame.qq.com</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>浪live</td>
        <td>play.lang.live & www.lang.live</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>花椒</td>
        <td>www.huajiao.com</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>抖音直播</td>
        <td>live.douyin.com</td>
        <td>✅ 支持</td>
        <td>✅ 支持</td>
    </tr>
    <tr align="center">
        <td>猫耳</td>
        <td>fm.missevan.com</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>克拉克拉</td>
        <td>www.hongdoufm.com</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>YY直播</td>
        <td>www.yy.com</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>微博直播</td>
        <td>weibo.com</td>
        <td>✅ 支持</td>
        <td></td>
    </tr>
    <tr align="center">
        <td>SOOP</td>
        <td>play.sooplive.com</td>
        <td>✅ 支持</td>
        <td>✅ 支持</td>
    </tr>
</table>

### cookie 在 config.yml 中的设置方法

cookie 的设置以域名为单位。比如想在录制抖音直播时使用 cookie，`config.yml` 中可以这样写：

```yaml
cookies:
  live.douyin.com: __ac_nonce=123456789012345678903;name=value
```

这里的 name 和 value 只是示例，多条键值对用分号隔开；具体需要哪些键取决于平台，请自行获取。

## 安装与运行

### 环境要求

- 64 位 Windows / macOS / Linux，或支持 Docker 的 NAS
- 可写磁盘空间：录制文件会持续增长，建议单独挂载数据盘
- 端口 `8080`（可在配置的 `rpc.bind` 修改）
- FFmpeg：不强制预装。程序会自动下载内置工具；也可在配置中指定 `ffmpeg_path`
- Go 源码运行需要 1.26+；普通用户直接用二进制或镜像即可

无论哪种方式，第一次启动前都建议先准备配置文件。最小配置至少包含监听地址、输出目录和一个直播间：

```yaml
rpc:
  enable: true
  bind: :8080
out_put_path: /path/to/Videos
live_rooms:
  - url: https://live.bilibili.com/123456
```

完整字段见仓库根目录的 `config.yml`（含注释）。也可以先启动，再在网页左侧 `设置` 中编辑。

### 二进制

| 系统 | 安装说明 |
|---|---|
| Windows | https://github.com/bililive-go/bililive-go/wiki/Install-Windows |
| macOS | https://github.com/bililive-go/bililive-go/wiki/Install-macOS |
| Linux | https://github.com/bililive-go/bililive-go/wiki/Install-Linux |

通用步骤：

1. 从 [Releases](https://github.com/bililive-go/bililive-go/releases) 下载对应平台的压缩包并解压
2. 把 `config.yml` 放在可执行文件同目录，或启动时用 `-c /path/to/config.yml` 指定
3. 启动程序，浏览器访问 `http://<主机>:8080`

```bash
chmod +x bililive-linux-amd64
./bililive-linux-amd64 -c ./config.yml
```

Linux 可用 `contrib/systemd/bililive-go.service` 配置开机自启（按文件内注释修改用户与路径）。

### Docker

镜像地址：`ghcr.io/bililive-go/bililive-go`。Docker Hub 的 `chigusa/bililive-go` 作为同步镜像保留。

> 镜像由 `.github/workflows/ghcr.yaml` 在推送 `master` 或版本 tag 时构建。GHCR 包默认私有，首次拉取前需 `docker login ghcr.io`，或在 GitHub 包设置中改为公开。

可用的 tag：

| tag | 含义 |
|---|---|
| `latest` | 最新正式版 |
| `v0.8.0` 等 | 固定版本，生产环境推荐 |
| `edge` | `master` 最新提交，可能不稳定 |
| `sha-<7位>` | 精确到提交，便于复现 |

```bash
docker login ghcr.io
docker run -d --restart=unless-stopped \
  --name bililive-go \
  -e PUID=$(id -u) -e PGID=$(id -g) -e UMASK=022 -e TZ=Asia/Shanghai \
  -v "$PWD/config.yml:/etc/bililive-go/config.yml" \
  -v "$PWD/Videos:/srv/bililive" \
  -p 8080:8080 \
  ghcr.io/bililive-go/bililive-go:latest
```

- `PUID` / `PGID`：容器以该 UID/GID 写入录制目录，避免宿主机产生 root 文件；用 `id -u`、`id -g` 获取
- `UMASK`：新建文件权限，默认 `022`
- `/srv/bililive`：录制输出与 `.appdata`（数据库、统计）都在此目录，必须持久化
- `/etc/bililive-go/config.yml`：配置文件挂载点
- 不要挂载整个仓库目录，也不要覆盖 `/opt/bililive/tools`（内置工具在镜像里）
- 升级：`docker pull` 后重建容器；配置与视频在挂载卷中不会丢失

### Docker Compose

仓库根目录的 `docker-compose.yml` 默认使用 GHCR 镜像，并预留了可选的 Prometheus / Grafana 段。

```bash
# 1. 按需修改 compose 与 config.docker.yml
# 2. 如启用 Grafana，复制 example.env 为 .env 并改掉默认账号密码
cp example.env .env
docker login ghcr.io
docker compose up -d
docker compose logs -f bililive-go
```

`docker compose` 默认把 `config.docker.yml` 挂到 `/etc/bililive-go/config.yml`、`Videos/` 挂到 `/srv/bililive`。首次运行后可在网页中继续改配置。

NAS 用户使用自带 GUI 创建 compose 项目的流程见[群晖图文说明](./docs/Synology-related.md#如何用-docker-compose-安装-bgo)。

### 安全与网络

程序本身**没有登录鉴权**，所有 API（含录播文件、`/api/metrics`）对内网开放。不要直接暴露到公网，建议放在带 Basic Auth 的反向代理（Caddy / Nginx / Traefik）后面，并只开放 `8080`。

容器默认监听所有网卡；如只给反代使用，可把端口绑定改成 `127.0.0.1:8080:8080`。启用 Grafana 时，`3000` 端口同样建议只绑定本机。

### 升级与备份

- 升级前备份 `config.yml` 和整个数据目录
- 固定版本时修改镜像 tag 或使用 digest，不要长期使用 `latest`
- 跨大版本升级先看 Releases 说明；数据库迁移只向前，回退需要恢复备份

## 网页管理

### 修改设置

左侧 `设置` 可在线编辑配置并保存，但保存成功不代表立即生效：部分配置需要停止后重新开始监控，部分需要重启程序。

### 播放器与文件

直播间行内 `文件` 链接可直接打开该房间的录播目录，也可通过左侧 `文件` 浏览。

### 主题与语言

顶栏右侧提供明亮/暗黑、中文/English 切换，选择保存在浏览器；日期与图表单位随语言切换。

## 内置监控面板

左侧 `监控面板` 是开箱即用的统计页，无需部署外部系统：

- 直播间开播、录制、监控状态，本次录制时长、输出文件大小、当前下载速率
- 下载速率曲线与窗口内开播时长排行（1 小时 ~ 30 天）
- 每日开播时长、单直播间开播历史（场次、起止时间、结束原因、按日时长）
- 系统内存、各子进程内存历史
- 5 秒 ~ 1 分钟自动刷新

原始 IO 统计（网络、磁盘、请求状态、内存明细）在 `IO 统计` 页查看，支持时间范围与聚合粒度。需要长期存储、跨实例聚合或告警时，仍可选配 Prometheus / Grafana：[docs/grafana.md](docs/grafana.md)。

![image](https://github.com/bililive-go/bililive-go/raw/master/docs/dashboard.webp)

## 转码与上传队列

左侧 `转码/上传队列` 展示：

- **等待转码**：已入队、尚未完成的文件
- **转码失败**：失败任务保留源文件，可逐条重试或清空队列
- **上传队列**：转码完成后自动入队，失败可重试，最近完成记录保留远程路径

上传在 `config.yml` 的 `remote_upload` 中配置：

```yaml
remote_upload:
  enable: true
  provider: s3            # webdav | s3
  path_tmpl: /录播归档/{{ .Platform }}/{{ .HostName }}/{{ .FileName }}
  delete_after_upload: false
  retry_limit: 3
  s3:
    endpoint: s3.example.com
    region: us-east-1
    bucket: recordings
    access_key: ""
    secret_key: ""
    path_style: false
    use_ssl: true
```

WebDAV 使用 `webdav.url`、`base_path`、`username`、`password`。上传任务与转码共享并发上限（`task_queue.max_concurrent`）。

## 通知

支持 Telegram、ntfy、邮件、Bark、WxPusher 等渠道，详见 [通知服务文档](docs/notify.md)。

## 常见问题

[docs/FAQ.md](docs/FAQ.md)

## 开发

支持 Windows、macOS、Linux 原生开发，无需 WSL。项目统一使用 npm 与 `go run ./build.go`。

### 前置要求

| 工具 | 版本要求 | 说明 |
|------|----------|------|
| [Go](https://golang.org/dl/) | 1.26+ | 后端开发语言 |
| [Node.js](https://nodejs.org/) | 20.10.0 | 前端构建，版本固定于 `.node-version` |
| npm | 随 Node.js 安装 | 前端包管理与脚本执行 |
| [golangci-lint](https://golangci-lint.run/) | 2.x | Go 静态检查 |
| [Git](https://git-scm.com/) | - | 版本控制 |
| [FFmpeg](https://ffmpeg.org/) | - | 可选，程序会自动下载 |

请使用 nvm / fnm 切换到 `.node-version` 指定的版本：

```bash
nvm use   # 或 fnm use
```

### 快速开始

```bash
git clone https://github.com/bililive-go/bililive-go.git
cd bililive-go

go generate ./tools/devtools.go
go run ./build.go build-web
go run ./build.go dev
```

运行 `bin/bililive-windows-amd64.exe`（Windows）或对应的 `bin/bililive-*`。

### 前端开发

```bash
npm --prefix src/webapp ci
npm --prefix src/webapp start   # API 代理到 127.0.0.1:8080
```

修改依赖用 `npm --prefix src/webapp install` 更新锁文件。根目录 `package-lock.json` 供 Playwright 使用，`src/webapp/package-lock.json` 供前端使用，两者都要提交。

### VSCode 调试

```bash
cp .vscode/launch.example.json .vscode/launch.json
cp .vscode/tasks.example.json .vscode/tasks.json
```

打开项目后按 `F5`，选择 **🚀 Debug Main Program**（增量编译，依赖 `tasks.json`）或 **Debug Main Program (Source)**。两个文件均被 gitignore，可自由修改。详见 [test/README.md](test/README.md)。

### 常用命令

| 功能 | 命令 |
|------|------|
| 查看帮助 | `go run ./build.go --help` |
| 安装前端依赖 | `npm --prefix src/webapp ci` |
| 启动前端开发服务器 | `npm --prefix src/webapp start` |
| 开发构建 | `go run ./build.go dev` |
| 发布构建 | `go run ./build.go release` |
| 构建前端 | `go run ./build.go build-web` |
| 运行测试 | `go run ./build.go test` |
| 代码检查 | `go run ./build.go lint` |
| E2E 测试 | `go run ./build.go test-e2e` |
| 清理产物 | `go run ./build.go clean` |

提交前建议执行：

```bash
npm --prefix src/webapp run build
go run ./build.go lint
go run ./build.go test
```

### CI / 镜像

仓库只保留一个工作流 `.github/workflows/ghcr.yaml`：推送 `master` 构建 `edge` 与 `sha-*` 镜像，推送 `v*` tag 构建同名镜像（非 `rc` 同时更新 `latest`），全部推送到 GHCR。合并前没有自动 lint/测试，请本地执行上面的检查。

### 项目结构

```
bililive-go/
├── src/
│   ├── cmd/           # 主程序与启动器
│   ├── configs/       # 配置管理
│   ├── live/          # 平台解析
│   ├── pkg/           # 通用包（含 remoteupload）
│   ├── pipeline/      # 后处理管道与队列
│   ├── recorders/     # 录制器
│   ├── servers/       # HTTP API
│   └── webapp/        # React 前端
├── test/              # 测试工具
├── tools/             # 开发工具依赖
├── config.yml         # 配置文件
└── build.go           # 构建入口
```

## Wiki

[Wiki](https://github.com/bililive-go/bililive-go/wiki)

## API

[API doc](https://github.com/bililive-go/bililive-go/blob/master/docs/API.md)

## 参考

- [you-get](https://github.com/soimort/you-get)
- [ykdl](https://github.com/zhangn1985/ykdl)
- [youtube-dl](https://github.com/ytdl-org/youtube-dl)
