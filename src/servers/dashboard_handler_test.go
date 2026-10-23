package servers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/bluele/gcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/bililive-go/bililive-go/src/instance"
	"github.com/bililive-go/bililive-go/src/listeners"
	"github.com/bililive-go/bililive-go/src/live"
	livemock "github.com/bililive-go/bililive-go/src/live/mock"
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
