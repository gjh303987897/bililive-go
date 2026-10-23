package servers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/bluele/gcache"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/bililive-go/bililive-go/src/configs"
	"github.com/bililive-go/bililive-go/src/instance"
	"github.com/bililive-go/bililive-go/src/listeners"
	"github.com/bililive-go/bililive-go/src/live"
	livemock "github.com/bililive-go/bililive-go/src/live/mock"
	"github.com/bililive-go/bililive-go/src/livestate"
	"github.com/bililive-go/bililive-go/src/pkg/iostats"
	"github.com/bililive-go/bililive-go/src/recorders"
	"github.com/bililive-go/bililive-go/src/types"
)

func TestParseDashboardRangeDefaults(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	r := httptest.NewRequest("GET", "/api/dashboard", nil)

	start, end := parseDashboardRange(r, now)

	assert.Equal(t, now, end)
	assert.Equal(t, now.Add(-dashboardDefaultRange), start)
}

func TestParseDashboardRangeClampsAndSwaps(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	start := now.Add(-200 * 24 * time.Hour).UnixMilli()
	end := now.Add(-time.Hour).UnixMilli()
	r := httptest.NewRequest("GET", "/api/dashboard?start_time="+itoa(start)+"&end_time="+itoa(end), nil)

	gotStart, gotEnd := parseDashboardRange(r, now)
	assert.Equal(t, dashboardMaxRange, gotEnd.Sub(gotStart))

	future := httptest.NewRequest("GET", "/api/dashboard?start_time="+itoa(now.Add(time.Hour).UnixMilli())+"&end_time="+itoa(now.Add(2*time.Hour).UnixMilli()), nil)
	gotStart, gotEnd = parseDashboardRange(future, now)
	assert.True(t, gotEnd.Equal(now))
	assert.False(t, gotStart.After(gotEnd))

	reversed := httptest.NewRequest("GET", "/api/dashboard?start_time="+itoa(now.Add(-time.Hour).UnixMilli())+"&end_time="+itoa(now.Add(-2*time.Hour).UnixMilli()), nil)
	gotStart, gotEnd = parseDashboardRange(reversed, now)
	assert.True(t, gotEnd.Equal(now.Add(-2*time.Hour)))
	assert.True(t, gotStart.Equal(gotEnd))
}

func TestGetDashboardReportsRoomState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	liveID := types.LiveID("live-1")
	liveObj := livemock.NewMockLive(ctrl)
	liveObj.EXPECT().GetLiveId().Return(liveID).AnyTimes()
	liveObj.EXPECT().GetRawUrl().Return("https://example.com/room").AnyTimes()
	liveObj.EXPECT().GetPlatformCNName().Return("示例平台").AnyTimes()
	liveObj.EXPECT().GetLastStartTime().Return(time.Now().Add(-2 * time.Hour)).AnyTimes()

	cache := gcache.New(1).LRU().Build()
	require.NoError(t, cache.Set(liveObj, &live.Info{
		Live:     liveObj,
		HostName: "主播",
		RoomName: "房间",
		Status:   true,
	}))

	inst := &instance.Instance{Cache: cache}
	inst.Lives.Set(liveID, liveObj)

	ctx := context.WithValue(context.Background(), instance.Key, inst)
	inst.ListenerManager = listeners.NewManager(ctx)
	inst.RecorderManager = recorders.NewManager(ctx)

	rec := httptest.NewRecorder()
	getDashboard(rec, httptest.NewRequest("GET", "/api/dashboard", nil).WithContext(ctx))

	require.Equal(t, 200, rec.Code)
	var resp dashboardResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	assert.Equal(t, 1, resp.Summary.Rooms)
	assert.Equal(t, 0, resp.Summary.Listening)
	assert.Equal(t, 1, resp.Summary.Live)
	assert.False(t, resp.IOStatsEnabled)
	require.Len(t, resp.Rooms, 1)
	assert.Equal(t, "主播", resp.Rooms[0].HostName)
	assert.Equal(t, "示例平台", resp.Rooms[0].Platform)
	assert.False(t, resp.Rooms[0].Listening)
	assert.True(t, resp.Rooms[0].Status)
	assert.Greater(t, resp.Rooms[0].LiveSeconds, int64(3600))
	assert.Empty(t, resp.Bitrate)
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}

func TestParseDashboardDays(t *testing.T) {
	assert.Equal(t, dashboardHistoryDefaultDays, parseDashboardDays(httptest.NewRequest("GET", "/api/dashboard/live/x/history", nil)))
	assert.Equal(t, 7, parseDashboardDays(httptest.NewRequest("GET", "/api/dashboard/live/x/history?days=7", nil)))
	assert.Equal(t, dashboardHistoryMaxDays, parseDashboardDays(httptest.NewRequest("GET", "/api/dashboard/live/x/history?days=9999", nil)))
	assert.Equal(t, dashboardHistoryDefaultDays, parseDashboardDays(httptest.NewRequest("GET", "/api/dashboard/live/x/history?days=-2", nil)))
}

func newDashboardRoomFixture(t *testing.T) (*instance.Instance, context.Context, types.LiveID, func()) {
	t.Helper()
	ctrl := gomock.NewController(t)
	liveID := types.LiveID("live-history")
	liveObj := livemock.NewMockLive(ctrl)
	liveObj.EXPECT().GetLiveId().Return(liveID).AnyTimes()
	liveObj.EXPECT().GetRawUrl().Return("https://example.com/history").AnyTimes()
	liveObj.EXPECT().GetPlatformCNName().Return("示例平台").AnyTimes()
	liveObj.EXPECT().GetLastStartTime().Return(time.Now().Add(-time.Hour)).AnyTimes()

	cache := gcache.New(1).LRU().Build()
	require.NoError(t, cache.Set(liveObj, &live.Info{
		Live:     liveObj,
		HostName: "主播",
		RoomName: "房间",
		Status:   true,
	}))

	store, err := livestate.NewSQLiteStore(t.TempDir() + "/lives.db")
	require.NoError(t, err)

	inst := &instance.Instance{Cache: cache, LiveStateStore: store}
	inst.Lives.Set(liveID, liveObj)

	ctx := context.WithValue(context.Background(), instance.Key, inst)
	inst.ListenerManager = listeners.NewManager(ctx)
	inst.RecorderManager = recorders.NewManager(ctx)

	return inst, ctx, liveID, func() {
		require.NoError(t, store.Close())
		ctrl.Finish()
	}
}

func TestGetDashboardAggregatesDailyDuration(t *testing.T) {
	_, ctx, liveID, cleanup := newDashboardRoomFixture(t)
	defer cleanup()

	inst := instance.GetInstance(ctx)
	store := inst.LiveStateStore.(livestate.Store)
	now := time.Now()
	_, err := store.StartSession(ctx, string(liveID), "主播", "房间", now.Add(-90*time.Minute))
	require.NoError(t, err)
	require.NoError(t, store.EndSession(ctx, string(liveID), now.Add(-30*time.Minute), livestate.EndReasonNormal))

	rec := httptest.NewRecorder()
	getDashboard(rec, httptest.NewRequest("GET", "/api/dashboard", nil).WithContext(ctx))
	require.Equal(t, 200, rec.Code)

	var resp dashboardResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Daily, 1)
	assert.InDelta(t, float64(3600), float64(resp.Daily[0].Seconds), 5)
	assert.Equal(t, 1, resp.Daily[0].Sessions)
	assert.Equal(t, 1, resp.Daily[0].Rooms)
	assert.Equal(t, now.Local().Format("2006-01-02"), resp.Daily[0].Date)
}

func TestGetDashboardRoomHistory(t *testing.T) {
	_, ctx, liveID, cleanup := newDashboardRoomFixture(t)
	defer cleanup()

	inst := instance.GetInstance(ctx)
	store := inst.LiveStateStore.(livestate.Store)
	now := time.Now()
	_, err := store.StartSession(ctx, string(liveID), "主播", "房间", now.Add(-26*time.Hour))
	require.NoError(t, err)
	require.NoError(t, store.EndSession(ctx, string(liveID), now.Add(-25*time.Hour), livestate.EndReasonNormal))
	_, err = store.StartSession(ctx, string(liveID), "主播", "房间", now.Add(-2*time.Hour))
	require.NoError(t, err)
	require.NoError(t, store.EndSession(ctx, string(liveID), now.Add(-time.Hour), livestate.EndReasonUserStop))
	_, err = store.StartSession(ctx, string(liveID), "主播", "房间", now.Add(-10*time.Minute))
	require.NoError(t, err)

	request := httptest.NewRequest("GET", "/api/dashboard/live/"+string(liveID)+"/history?days=2", nil).WithContext(ctx)
	request = mux.SetURLVars(request, map[string]string{"id": string(liveID)})
	rec := httptest.NewRecorder()
	getDashboardRoomHistory(rec, request)
	require.Equal(t, 200, rec.Code)

	var resp dashboardRoomHistory
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, string(liveID), resp.LiveID)
	assert.Equal(t, 2, resp.Days)
	assert.Equal(t, 3, resp.TotalSessions)
	require.Len(t, resp.Sessions, 3)

	ongoing := resp.Sessions[0]
	assert.True(t, ongoing.Ongoing)
	assert.Empty(t, ongoing.EndReason)
	assert.LessOrEqual(t, ongoing.DurationSeconds, int64(10*60))

	var total int64
	for _, session := range resp.Sessions {
		total += session.DurationSeconds
	}
	assert.Equal(t, total, resp.TotalSeconds)
	require.Len(t, resp.Daily, 2)
	assert.Equal(t, int64(3600), resp.Daily[0].Seconds)
	assert.GreaterOrEqual(t, resp.Daily[1].Seconds, int64(3600))
	for _, point := range resp.Daily {
		assert.Equal(t, 1, point.Rooms)
	}
}

func TestGetDashboardRoomHistoryNotFound(t *testing.T) {
	_, ctx, _, cleanup := newDashboardRoomFixture(t)
	defer cleanup()

	request := httptest.NewRequest("GET", "/api/dashboard/live/missing/history", nil).WithContext(ctx)
	request = mux.SetURLVars(request, map[string]string{"id": "missing"})
	rec := httptest.NewRecorder()
	getDashboardRoomHistory(rec, request)
	assert.Equal(t, 404, rec.Code)
}

func TestGetDashboardRoomHistoryWithoutStore(t *testing.T) {
	_, ctx, liveID, cleanup := newDashboardRoomFixture(t)
	defer cleanup()

	instance.GetInstance(ctx).LiveStateStore = nil
	request := httptest.NewRequest("GET", "/api/dashboard/live/"+string(liveID)+"/history", nil).WithContext(ctx)
	request = mux.SetURLVars(request, map[string]string{"id": string(liveID)})
	rec := httptest.NewRecorder()
	getDashboardRoomHistory(rec, request)
	assert.Equal(t, 503, rec.Code)
}

func TestQueryDashboardBitrateReturnsMillisecondTimestamps(t *testing.T) {
	previous := configs.GetCurrentConfig()
	defer configs.SetCurrentConfig(previous)

	cfg := configs.NewConfig()
	cfg.AppDataPath = t.TempDir()
	configs.SetCurrentConfig(cfg)

	module, err := iostats.NewModule(context.Background(), iostats.DefaultConfig())
	require.NoError(t, err)
	statsModule, ok := module.(*iostats.Module)
	require.True(t, ok)
	defer statsModule.Close(context.Background())

	now := time.Now()
	start := now.Add(-time.Hour)
	for i := 0; i < 5; i++ {
		require.NoError(t, statsModule.GetStore().SaveIOStat(context.Background(), &iostats.IOStat{
			Timestamp:  now.Add(-time.Duration(i) * time.Minute).UnixMilli(),
			StatType:   iostats.StatTypeNetworkDownload,
			LiveID:     "live-1",
			Platform:   "示例平台",
			Speed:      int64(1000 * (i + 1)),
			TotalBytes: 0,
		}))
	}

	inst := &instance.Instance{IOStatsModule: statsModule}
	ctx := context.WithValue(context.Background(), instance.Key, inst)

	speeds, series, enabled := queryDashboardBitrate(ctx, inst, start, now, map[string]dashboardRoom{
		"live-1": {LiveID: "live-1", HostName: "主播"},
	})
	require.True(t, enabled)
	require.Len(t, series, 1)
	require.NotEmpty(t, series[0].Points)
	assert.Equal(t, "主播", series[0].HostName)
	assert.Equal(t, int64(1000), speeds["live-1"])
	for _, point := range series[0].Points {
		assert.Greater(t, point.T, int64(1_000_000_000_000), "时间戳应为毫秒")
		assert.GreaterOrEqual(t, point.T, start.UnixMilli()-int64(time.Minute/time.Millisecond))
		assert.LessOrEqual(t, point.T, now.UnixMilli())
	}
}
