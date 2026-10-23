package servers

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/bililive-go/bililive-go/src/instance"
	"github.com/bililive-go/bililive-go/src/livestate"
	"github.com/bililive-go/bililive-go/src/pkg/iostats"
	"github.com/bililive-go/bililive-go/src/recorders"
	"github.com/bililive-go/bililive-go/src/types"
)

const (
	// dashboardDefaultRange 默认统计窗口
	dashboardDefaultRange = 24 * time.Hour
	// dashboardMaxRange 允许的最大统计窗口，避免拉爆 SQLite 查询
	dashboardMaxRange = 90 * 24 * time.Hour
	// dashboardMaxPoints 单条曲线的最大点数，超出后服务端降采样
	dashboardMaxPoints = 400
	// dashboardMaxSeries 返回的码率曲线数量上限（按流量取前 N）
	dashboardMaxSeries = 20
	// dashboardSessionLimit 每个直播间读取的会话条数上限
	dashboardSessionLimit = 2000
)

// dashboardPoint 码率时间序列点，t 为毫秒时间戳，v 为字节/秒
type dashboardPoint struct {
	T int64 `json:"t"`
	V int64 `json:"v"`
}

type dashboardRoom struct {
	LiveID             string `json:"live_id"`
	Platform           string `json:"platform"`
	HostName           string `json:"host_name"`
	RoomName           string `json:"room_name"`
	LiveURL            string `json:"live_url"`
	Status             bool   `json:"status"`
	Listening          bool   `json:"listening"`
	Recording          bool   `json:"recording"`
	RecordingPreparing bool   `json:"recording_preparing"`
	LiveSeconds        int64  `json:"live_seconds"`
	RecordedSeconds    int64  `json:"recorded_seconds"`
	TotalBytes         int64  `json:"total_bytes"`
	Speed              int64  `json:"speed"`
	SecondsInRange     int64  `json:"seconds_in_range"`
	SessionsInRange    int    `json:"sessions_in_range"`
}

type dashboardSeries struct {
	LiveID   string           `json:"live_id"`
	HostName string           `json:"host_name"`
	RoomName string           `json:"room_name"`
	Points   []dashboardPoint `json:"points"`
}

type dashboardSummary struct {
	Rooms      int   `json:"rooms"`
	Listening  int   `json:"listening"`
	Live       int   `json:"live"`
	Recording  int   `json:"recording"`
	TotalBytes int64 `json:"total_bytes"`
}

type dashboardResponse struct {
	GeneratedAt    int64             `json:"generated_at"`
	StartTime      int64             `json:"start_time"`
	EndTime        int64             `json:"end_time"`
	IOStatsEnabled bool              `json:"iostats_enabled"`
	Summary        dashboardSummary  `json:"summary"`
	Rooms          []dashboardRoom   `json:"rooms"`
	Bitrate        []dashboardSeries `json:"bitrate"`
	DurationRank   []dashboardRoom   `json:"duration_ranking"`
}

// getDashboard 返回监控面板所需的聚合快照。
// Grafana 的四个面板（直播状态、30 天播放时长、输出文件大小、码率）全部由本接口提供，
// 数据源是进程内状态 + 会话历史 + IO 统计，不依赖 Prometheus。
func getDashboard(writer http.ResponseWriter, r *http.Request) {
	inst := instance.GetInstance(r.Context())
	if inst == nil {
		writeJsonWithStatusCode(writer, http.StatusServiceUnavailable, commonResp{
			ErrNo:  http.StatusServiceUnavailable,
			ErrMsg: "instance 未初始化",
		})
		return
	}

	now := time.Now()
	start, end := parseDashboardRange(r, now)

	ctx := r.Context()
	roomList := inst.Lives.Snapshot()

	rooms := make([]dashboardRoom, 0, len(roomList))
	lookup := make(map[string]dashboardRoom, len(roomList))
	summary := dashboardSummary{Rooms: len(roomList)}
	for id, l := range roomList {
		info := parseInfo(ctx, l)
		room := dashboardRoom{
			LiveID:             string(id),
			Platform:           l.GetPlatformCNName(),
			HostName:           info.HostName,
			RoomName:           info.RoomName,
			LiveURL:            l.GetRawUrl(),
			Status:             info.Status,
			Listening:          info.Listening,
			Recording:          info.Recording,
			RecordingPreparing: info.RecordingPreparing,
		}

		if info.Status {
			lastStart := l.GetLastStartTime()
			if !lastStart.IsZero() && now.After(lastStart) {
				room.LiveSeconds = int64(now.Sub(lastStart).Seconds())
			}
			summary.Live++
		}
		if info.Listening {
			summary.Listening++
		}
		if info.Recording || info.RecordingPreparing {
			room.TotalBytes, room.RecordedSeconds = dashboardRecorderState(ctx, inst, types.LiveID(id), now)
			if info.Recording {
				summary.Recording++
			}
			summary.TotalBytes += room.TotalBytes
		}

		room.SecondsInRange, room.SessionsInRange = dashboardSessionStats(ctx, inst, room.LiveID, start, end)
		rooms = append(rooms, room)
		lookup[room.LiveID] = room
	}

	speeds, series, iostatsEnabled := queryDashboardBitrate(ctx, inst, start, end, lookup)
	for i := range rooms {
		rooms[i].Speed = speeds[rooms[i].LiveID]
	}

	sort.Slice(rooms, func(i, j int) bool {
		a, b := rooms[i], rooms[j]
		if a.Recording != b.Recording {
			return a.Recording
		}
		if a.Status != b.Status {
			return a.Status
		}
		if a.Listening != b.Listening {
			return a.Listening
		}
		if a.Platform != b.Platform {
			return a.Platform < b.Platform
		}
		return a.LiveID < b.LiveID
	})

	ranking := make([]dashboardRoom, 0, len(rooms))
	for _, room := range rooms {
		if room.SecondsInRange > 0 {
			ranking = append(ranking, room)
		}
	}
	sort.Slice(ranking, func(i, j int) bool {
		return ranking[i].SecondsInRange > ranking[j].SecondsInRange
	})

	writeJSON(writer, dashboardResponse{
		GeneratedAt:    now.UnixMilli(),
		StartTime:      start.UnixMilli(),
		EndTime:        end.UnixMilli(),
		IOStatsEnabled: iostatsEnabled,
		Summary:        summary,
		Rooms:          rooms,
		Bitrate:        series,
		DurationRank:   ranking,
	})
}

// parseDashboardRange 解析查询窗口，缺省为最近 24 小时，最大 90 天。
func parseDashboardRange(r *http.Request, now time.Time) (time.Time, time.Time) {
	end := now
	if raw := r.URL.Query().Get("end_time"); raw != "" {
		if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
			end = time.UnixMilli(ms)
		}
	}
	start := end.Add(-dashboardDefaultRange)
	if raw := r.URL.Query().Get("start_time"); raw != "" {
		if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
			start = time.UnixMilli(ms)
		}
	}
	if end.After(now) {
		end = now
	}
	if start.After(end) {
		start = end
	}
	if end.Sub(start) > dashboardMaxRange {
		start = end.Add(-dashboardMaxRange)
	}
	return start, end
}

// dashboardRecorderState 读取当前录制文件大小与已录制时长。
func dashboardRecorderState(ctx context.Context, inst *instance.Instance, id types.LiveID, now time.Time) (totalBytes, recordedSeconds int64) {
	manager, ok := inst.RecorderManager.(recorders.Manager)
	if !ok {
		return 0, 0
	}
	recorder, err := manager.GetRecorder(ctx, id)
	if err != nil {
		return 0, 0
	}
	start := recorder.StartTime()
	if !start.IsZero() && now.After(start) {
		recordedSeconds = int64(now.Sub(start).Seconds())
	}
	status, err := recorder.GetStatus()
	if err != nil {
		return 0, recordedSeconds
	}
	totalBytes = parseRecorderSize(status["file_size"])
	if totalBytes == 0 {
		totalBytes = parseRecorderSize(status["total_size"])
	}
	return totalBytes, recordedSeconds
}

func parseRecorderSize(value any) int64 {
	raw, ok := value.(string)
	if !ok {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// dashboardSessionStats 统计窗口内的累计开播秒数与会话次数。
func dashboardSessionStats(ctx context.Context, inst *instance.Instance, liveID string, start, end time.Time) (seconds int64, count int) {
	store, ok := inst.LiveStateStore.(livestate.Store)
	if !ok {
		return 0, 0
	}
	sessions, err := store.GetSessionsByLiveID(ctx, liveID, dashboardSessionLimit)
	if err != nil {
		return 0, 0
	}
	for _, session := range sessions {
		sessionStart := session.StartTime
		if sessionStart.IsZero() || sessionStart.After(end) {
			continue
		}
		sessionEnd := session.EndTime
		if sessionEnd.IsZero() || sessionEnd.After(end) {
			sessionEnd = end
		}
		if sessionStart.Before(start) {
			sessionStart = start
		}
		if sessionEnd.After(sessionStart) {
			seconds += int64(sessionEnd.Sub(sessionStart).Seconds())
			count++
		}
	}
	return seconds, count
}

// queryDashboardBitrate 查询窗口内的下载速率，返回最新速度、降采样后的曲线和统计模块是否可用。
func queryDashboardBitrate(ctx context.Context, inst *instance.Instance, start, end time.Time, lookup map[string]dashboardRoom) (map[string]int64, []dashboardSeries, bool) {
	speeds := make(map[string]int64)
	module, ok := inst.IOStatsModule.(*iostats.Module)
	if !ok || module == nil {
		return speeds, nil, false
	}

	aggregation := "hour"
	if end.Sub(start) <= 6*time.Hour {
		aggregation = "minute"
	}
	stats, err := module.GetStore().QueryIOStats(ctx, iostats.IOStatsQuery{
		StartTime:   start.UnixMilli(),
		EndTime:     end.UnixMilli(),
		StatTypes:   []iostats.StatType{iostats.StatTypeNetworkDownload},
		Aggregation: aggregation,
	})
	if err != nil {
		return speeds, nil, true
	}

	// 最新速率：窗口末尾 1 分钟内的最后一个采样点
	speedCutoff := end.Add(-time.Minute).UnixMilli()
	buckets := make(map[string]map[int64]*struct {
		sum   int64
		count int
	})
	for _, stat := range stats {
		if stat.LiveID == "" {
			continue
		}
		if stat.Timestamp >= speedCutoff {
			speeds[stat.LiveID] = stat.Speed
		}
		if buckets[stat.LiveID] == nil {
			buckets[stat.LiveID] = make(map[int64]*struct {
				sum   int64
				count int
			})
		}
		bucket := stat.Timestamp / dashboardBucketMillis(start, end)
		entry := buckets[stat.LiveID][bucket]
		if entry == nil {
			entry = &struct {
				sum   int64
				count int
			}{}
			buckets[stat.LiveID][bucket] = entry
		}
		entry.sum += stat.Speed
		entry.count++
	}

	series := make([]dashboardSeries, 0, len(buckets))
	for liveID, bucket := range buckets {
		room := lookup[liveID]
		item := dashboardSeries{LiveID: liveID, HostName: room.HostName, RoomName: room.RoomName}
		if item.HostName == "" {
			item.HostName = liveID
		}
		item.Points = make([]dashboardPoint, 0, len(bucket))
		for ts, entry := range bucket {
			if entry.count == 0 {
				continue
			}
			item.Points = append(item.Points, dashboardPoint{T: ts, V: entry.sum / int64(entry.count)})
		}
		sort.Slice(item.Points, func(i, j int) bool { return item.Points[i].T < item.Points[j].T })
		if len(item.Points) > 0 {
			series = append(series, item)
		}
	}
	sort.Slice(series, func(i, j int) bool {
		return seriesSum(series[i]) > seriesSum(series[j])
	})
	if len(series) > dashboardMaxSeries {
		series = series[:dashboardMaxSeries]
	}
	return speeds, series, true
}

func seriesSum(s dashboardSeries) int64 {
	var total int64
	for _, p := range s.Points {
		total += p.V
	}
	return total
}

// dashboardBucketMillis 计算降采样桶大小，保证点数不超过 dashboardMaxPoints。
func dashboardBucketMillis(start, end time.Time) int64 {
	span := end.Sub(start).Milliseconds()
	bucket := span / dashboardMaxPoints
	if bucket < int64(time.Minute/time.Millisecond) {
		bucket = int64(time.Minute / time.Millisecond)
	}
	return bucket
}
