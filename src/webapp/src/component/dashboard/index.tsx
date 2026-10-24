import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Drawer,
  Empty,
  Row,
  Select,
  Space,
  Spin,
  Statistic,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import {
  AreaChartOutlined,
  DatabaseOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip as RechartsTooltip,
  XAxis,
  YAxis,
} from 'recharts';
import dayjs from 'dayjs';
import MemoryStats from '../io-stats/MemoryStats';
import MemoryHistoryChart from '../io-stats/MemoryHistoryChart';
import { useI18n } from '../../i18n';
import './index.css';

const { Text } = Typography;

interface DashboardPoint {
  t: number;
  v: number;
}

interface DashboardRoom {
  live_id: string;
  platform: string;
  host_name: string;
  room_name: string;
  live_url: string;
  status: boolean;
  listening: boolean;
  recording: boolean;
  recording_preparing: boolean;
  live_seconds: number;
  recorded_seconds: number;
  total_bytes: number;
  speed: number;
  seconds_in_range: number;
  sessions_in_range: number;
}

interface DashboardSeries {
  live_id: string;
  host_name: string;
  room_name: string;
  points: DashboardPoint[];
}

interface DashboardData {
  generated_at: number;
  start_time: number;
  end_time: number;
  iostats_enabled: boolean;
  summary: {
    rooms: number;
    listening: number;
    live: number;
    recording: number;
    total_bytes: number;
  };
  rooms: DashboardRoom[];
  bitrate: DashboardSeries[];
  duration_ranking: DashboardRoom[];
  daily: DashboardDailyPoint[];
}

interface DashboardDailyPoint {
  date: string;
  seconds: number;
  sessions: number;
  rooms: number;
}

interface RoomHistorySession {
  start_time: number;
  end_time: number;
  duration_seconds: number;
  ongoing: boolean;
  end_reason: string;
}

interface RoomHistory {
  live_id: string;
  host_name: string;
  room_name: string;
  platform: string;
  days: number;
  total_seconds: number;
  total_sessions: number;
  sessions: RoomHistorySession[];
  daily: DashboardDailyPoint[];
}

const END_REASON_KEYS: Record<string, string> = {
  normal: 'dash.reason.normal',
  user_stop: 'dash.reason.user_stop',
  crash: 'dash.reason.crash',
  error: 'dash.reason.error',
  unknown: 'dash.reason.unknown',
};

const RANGE_KEYS = [
  { value: 60 * 60 * 1000, key: 'dash.range1h' },
  { value: 6 * 60 * 60 * 1000, key: 'dash.range6h' },
  { value: 24 * 60 * 60 * 1000, key: 'dash.range24h' },
  { value: 7 * 24 * 60 * 60 * 1000, key: 'dash.range7d' },
  { value: 30 * 24 * 60 * 60 * 1000, key: 'dash.range30d' },
];

const REFRESH_KEYS = [
  { value: 5000, label: '5s' },
  { value: 10000, label: '10s' },
  { value: 30000, label: '30s' },
  { value: 60000, label: '1m' },
  { value: 0, key: 'dash.off' },
];

const SERIES_COLORS = ['#1677ff', '#52c41a', '#fa8c16', '#eb2f96', '#722ed1', '#13c2c2', '#f5222d', '#a0d911', '#2f54eb', '#faad14'];

const formatBytes = (value: number): string => {
  if (!value || value <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let size = value;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit++;
  }
  return `${size.toFixed(size >= 100 || unit === 0 ? 0 : 2)} ${units[unit]}`;
};

const formatSpeed = (value: number): string => `${formatBytes(value)}/s`;

const formatDuration = (seconds: number, en = false): string => {
  if (!seconds || seconds <= 0) return '-';
  const total = Math.floor(seconds);
  const days = Math.floor(total / 86400);
  const hours = Math.floor((total % 86400) / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  if (days > 0) return en ? `${days}d ${hours}h` : `${days} 天 ${hours} 小时`;
  if (hours > 0) return en ? `${hours}h ${minutes}m` : `${hours} 小时 ${minutes} 分`;
  if (minutes > 0) return en ? `${minutes}m ${secs}s` : `${minutes} 分 ${secs} 秒`;
  return en ? `${secs}s` : `${secs} 秒`;
};

const roomLabel = (room: Pick<DashboardRoom, 'host_name' | 'room_name' | 'live_id'>): string =>
  room.host_name || room.room_name || room.live_id;

const Dashboard: React.FC = () => {
  const { t, locale } = useI18n();
  const en = locale === 'en';
  const [range, setRange] = useState(RANGE_KEYS[2].value);
  const [refreshInterval, setRefreshInterval] = useState(10000);
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [updatedAt, setUpdatedAt] = useState<number>(0);
  const [historyRoom, setHistoryRoom] = useState<DashboardRoom | null>(null);
  const [history, setHistory] = useState<RoomHistory | null>(null);
  const [historyLoading, setHistoryLoading] = useState(false);
  const inFlight = useRef(false);
  const historyDays = Math.max(1, Math.min(365, Math.round(range / 86400000)));

  const load = useCallback(async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    setLoading(true);
    try {
      const end = Date.now();
      const start = end - range;
      const response = await fetch(`/api/dashboard?start_time=${start}&end_time=${end}`);
      const payload = await response.json();
      if (!response.ok || payload.err_no) {
        throw new Error(payload.err_msg || `HTTP ${response.status}`);
      }
      setData(payload as DashboardData);
      setUpdatedAt(Date.now());
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
      inFlight.current = false;
    }
  }, [range]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (!refreshInterval) return undefined;
    const timer = window.setInterval(() => {
      if (!document.hidden) load();
    }, refreshInterval);
    return () => window.clearInterval(timer);
  }, [refreshInterval, load]);

  const bitrateChart = useMemo(() => {
    if (!data) return { rows: [] as Array<Record<string, number>>, keys: [] as DashboardSeries[] };
    const rows = new Map<number, Record<string, number>>();
    data.bitrate.forEach((series) => {
      series.points.forEach((point) => {
        const row = rows.get(point.t) ?? ({ t: point.t } as Record<string, number>);
        row[series.live_id] = point.v;
        rows.set(point.t, row);
      });
    });
    return {
      rows: Array.from(rows.values()).sort((a, b) => a.t - b.t),
      keys: data.bitrate,
    };
  }, [data]);

  const rankingChart = useMemo(() => {
    if (!data) return [];
    return data.duration_ranking.slice(0, 15).map((room) => ({
      name: roomLabel(room),
      hours: Number((room.seconds_in_range / 3600).toFixed(2)),
    }));
  }, [data]);

  const dailyChart = useMemo(() => (data?.daily ?? []).map((point) => ({
    date: point.date.slice(5),
    hours: Number((point.seconds / 3600).toFixed(2)),
    sessions: point.sessions,
    rooms: point.rooms,
  })), [data]);

  useEffect(() => {
    if (!historyRoom) return undefined;
    let cancelled = false;
    setHistory(null);
    setHistoryLoading(true);
    (async () => {
      try {
        const response = await fetch(
          `/api/dashboard/live/${encodeURIComponent(historyRoom.live_id)}/history?days=${historyDays}&limit=100`,
        );
        const payload = await response.json();
        if (!response.ok || payload.err_no) {
          throw new Error(payload.err_msg || `HTTP ${response.status}`);
        }
        if (!cancelled) setHistory(payload as RoomHistory);
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      } finally {
        if (!cancelled) setHistoryLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [historyRoom, historyDays]);

  const historyColumns = useMemo<ColumnsType<RoomHistorySession>>(() => [
    {
      title: t('dash.startAt'),
      dataIndex: 'start_time',
      key: 'start_time',
      width: 160,
      render: (value: number) => dayjs(value).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: t('dash.endAt'),
      dataIndex: 'end_time',
      key: 'end_time',
      width: 160,
      render: (value: number, session) => (session.ongoing ? <Tag color="green">{t('dash.liveNow')}</Tag> : dayjs(value).format('YYYY-MM-DD HH:mm')),
    },
    {
      title: t('dash.duration'),
      dataIndex: 'duration_seconds',
      key: 'duration_seconds',
      width: 110,
      render: (value: number) => formatDuration(value, en),
    },
    {
      title: t('dash.endReason'),
      dataIndex: 'end_reason',
      key: 'end_reason',
      render: (value: string, session) => (session.ongoing ? '-' : t(END_REASON_KEYS[value] ?? 'dash.reason.unknown')),
    },
  ], [t, en]);

  const columns = useMemo<ColumnsType<DashboardRoom>>(() => [
    {
      title: t('queue.status'),
      key: 'status',
      width: 130,
      render: (_, room) => {
        const tags: React.ReactNode[] = [];
        if (room.recording) tags.push(<Tag color="red" key="rec">{t('dash.recording')}</Tag>);
        if (room.recording_preparing) tags.push(<Tag color="orange" key="prep">{t('dash.preparing')}</Tag>);
        if (room.status) tags.push(<Tag color="green" key="live">{t('dash.live')}</Tag>);
        if (room.listening) tags.push(<Tag color="blue" key="listen">{t('dash.listening')}</Tag>);
        if (tags.length === 0) tags.push(<Tag key="idle">{t('dash.idle')}</Tag>);
        return <Space size={4} wrap>{tags}</Space>;
      },
    },
    {
      title: t('dash.platform'),
      dataIndex: 'platform',
      key: 'platform',
      width: 90,
    },
    {
      title: t('dash.hostRoom'),
      key: 'room',
      render: (_, room) => (
        <div>
          <Button type="link" className="dashboard-room-link" onClick={() => setHistoryRoom(room)}>
            {roomLabel(room)}
          </Button>
          <div className="dashboard-room-sub">
            {room.room_name}
            {' · '}
            <Link to={`/?room=${room.live_id}`}>{t('dash.openRoom')}</Link>
          </div>
        </div>
      ),
    },
    {
      title: t('dash.liveDuration'),
      dataIndex: 'live_seconds',
      key: 'live_seconds',
      width: 110,
      sorter: (a, b) => a.live_seconds - b.live_seconds,
      render: (value: number) => formatDuration(value, en),
    },
    {
      title: t('dash.recordDuration'),
      dataIndex: 'recorded_seconds',
      key: 'recorded_seconds',
      width: 110,
      sorter: (a, b) => a.recorded_seconds - b.recorded_seconds,
      render: (value: number) => formatDuration(value, en),
    },
    {
      title: t('dash.fileSize'),
      dataIndex: 'total_bytes',
      key: 'total_bytes',
      width: 110,
      sorter: (a, b) => a.total_bytes - b.total_bytes,
      render: (value: number) => formatBytes(value),
    },
    {
      title: t('dash.speed'),
      dataIndex: 'speed',
      key: 'speed',
      width: 110,
      sorter: (a, b) => a.speed - b.speed,
      render: (value: number) => formatSpeed(value),
    },
    {
      title: t('dash.inWindow'),
      dataIndex: 'seconds_in_range',
      key: 'seconds_in_range',
      width: 130,
      sorter: (a, b) => a.seconds_in_range - b.seconds_in_range,
      render: (value: number, room) => (
        <Tooltip title={t('dash.sessionsTooltip', { n: room.sessions_in_range })}>{formatDuration(value, en)}</Tooltip>
      ),
    },
  ], [t, en]);

  const summary = data?.summary;

  return (
    <div className="dashboard-page">
      <Card size="small" className="dashboard-toolbar">
        <Space wrap>
          <Text strong>{t('dash.window')}</Text>
          <Select
            value={range}
            style={{ width: 140 }}
            options={RANGE_KEYS.map((item) => ({ value: item.value, label: t(item.key) }))}
            onChange={setRange}
          />
          <Text strong>{t('dash.refresh')}</Text>
          <Select
            value={refreshInterval}
            style={{ width: 110 }}
            options={REFRESH_KEYS.map((item) => ({ value: item.value, label: item.key ? t(item.key) : item.label }))}
            onChange={setRefreshInterval}
          />
          <Button icon={<ReloadOutlined />} loading={loading} onClick={load}>{t('common.refresh')}</Button>
          <Link to="/iostats">
            <Button icon={<DatabaseOutlined />} type="link">{t('dash.rawIO')}</Button>
          </Link>
          <Text type="secondary">
            {t('common.updatedAt')} {updatedAt ? dayjs(updatedAt).format('HH:mm:ss') : '-'}
          </Text>
        </Space>
      </Card>

      {error && (
        <Alert
          type="error"
          showIcon
          closable
          message={t('dash.loadFailed')}
          description={error}
          onClose={() => setError(null)}
          className="dashboard-alert"
        />
      )}

      <Spin spinning={loading}>
        <Row gutter={16} className="dashboard-summary">
          <Col xs={12} md={6}>
            <Card size="small">
              <Statistic title={t('dash.rooms')} value={summary?.rooms ?? 0} prefix={<VideoCameraOutlined />} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size="small">
              <Statistic title={t('dash.monitoring')} value={summary?.listening ?? 0} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size="small">
              <Statistic title={t('dash.liveRecording')} value={summary?.live ?? 0} suffix={`/ ${summary?.recording ?? 0}`} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size="small">
              <Statistic title={t('dash.totalBytes')} value={formatBytes(summary?.total_bytes ?? 0)} />
            </Card>
          </Col>
        </Row>

        <Tabs
          defaultActiveKey="overview"
          items={[
            {
              key: 'overview',
              label: <span><ThunderboltOutlined />{t('dash.overview')}</span>,
              children: (
                <>
                  <Card size="small" title={t('dash.bitrate')} className="dashboard-card">
                    {bitrateChart.rows.length === 0 ? (
                      <Empty
                        description={data?.iostats_enabled === false ? t('dash.ioDisabled') : t('dash.bitrateEmpty')}
                      />
                    ) : (
                      <div className="dashboard-chart">
                        <ResponsiveContainer>
                          <LineChart data={bitrateChart.rows}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis
                              dataKey="t"
                              tickFormatter={(value) => dayjs(Number(value)).format(range > 86400000 ? 'MM-DD HH:mm' : 'HH:mm')}
                            />
                            <YAxis tickFormatter={(value) => formatSpeed(Number(value))} width={90} />
                            <RechartsTooltip
                              labelFormatter={(value) => dayjs(Number(value)).format('YYYY-MM-DD HH:mm:ss')}
                              formatter={(value, name) => {
                                const speed = typeof value === 'number' ? value : Number(value);
                                const series = bitrateChart.keys.find((item) => item.live_id === name);
                                return [formatSpeed(speed), series ? roomLabel(series) : String(name)];
                              }}
                            />
                            <Legend
                              formatter={(value: string) => {
                                const series = bitrateChart.keys.find((item) => item.live_id === value);
                                return series ? roomLabel(series) : value;
                              }}
                            />
                            {bitrateChart.keys.map((series, index) => (
                              <Line
                                key={series.live_id}
                                type="monotone"
                                dataKey={series.live_id}
                                stroke={SERIES_COLORS[index % SERIES_COLORS.length]}
                                dot={false}
                                connectNulls
                                isAnimationActive={false}
                              />
                            ))}
                          </LineChart>
                        </ResponsiveContainer>
                      </div>
                    )}
                  </Card>

                  <Card size="small" title={t('dash.rank')} className="dashboard-card">
                    {rankingChart.length === 0 ? (
                      <Empty description={t('dash.rankEmpty')} />
                    ) : (
                      <div className="dashboard-chart dashboard-chart-small">
                        <ResponsiveContainer>
                          <BarChart data={rankingChart} layout="vertical" margin={{ left: 24 }}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis type="number" unit="h" />
                            <YAxis type="category" dataKey="name" width={120} />
                            <RechartsTooltip formatter={(value) => [en ? `${Number(value)}h` : `${Number(value)} 小时`, t('dash.liveDuration')]} />
                            <Bar dataKey="hours" fill="#1677ff" isAnimationActive={false} />
                          </BarChart>
                        </ResponsiveContainer>
                      </div>
                    )}
                  </Card>

                  <Card size="small" title={t('dash.daily')} className="dashboard-card">
                    {dailyChart.length === 0 ? (
                      <Empty description={t('dash.rankEmpty')} />
                    ) : (
                      <div className="dashboard-chart dashboard-chart-small">
                        <ResponsiveContainer>
                          <BarChart data={dailyChart}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis dataKey="date" />
                            <YAxis unit="h" />
                            <RechartsTooltip
                              formatter={(value, name, entry) => {
                                const point = entry?.payload as { sessions: number; rooms: number } | undefined;
                                return [
                                  en ? `${Number(value)}h` : `${Number(value)} 小时`,
                                  en
                                    ? `${point?.sessions ?? 0} sessions · ${point?.rooms ?? 0} rooms`
                                    : `${point?.sessions ?? 0} 场 · ${point?.rooms ?? 0} 个直播间`,
                                ];
                              }}
                            />
                            <Bar dataKey="hours" fill="#52c41a" isAnimationActive={false} />
                          </BarChart>
                        </ResponsiveContainer>
                      </div>
                    )}
                  </Card>

                  <Card size="small" title={t('dash.details')} className="dashboard-card">
                    <Table
                      size="small"
                      rowKey="live_id"
                      dataSource={data?.rooms ?? []}
                      columns={columns}
                      pagination={{ pageSize: 20, hideOnSinglePage: true }}
                      scroll={{ x: 900 }}
                    />
                  </Card>
                </>
              ),
            },
            {
              key: 'system',
              label: <span><AreaChartOutlined />{t('dash.system')}</span>,
              children: (
                <>
                  <MemoryStats />
                  <div className="dashboard-chart">
                    <MemoryHistoryChart
                      startTime={data?.start_time ?? Date.now() - range}
                      endTime={data?.end_time ?? Date.now()}
                    />
                  </div>
                </>
              ),
            },
          ]}
        />
      </Spin>

      <Drawer
        width={720}
        open={historyRoom !== null}
        onClose={() => setHistoryRoom(null)}
        title={historyRoom ? t('dash.historyOf', { name: roomLabel(historyRoom) }) : t('dash.history')}
        extra={historyRoom && (
          <Link to={`/?room=${historyRoom.live_id}`}>
            <Button type="link">{t("dash.openRoom")}</Button>
          </Link>
        )}
      >
        <Spin spinning={historyLoading}>
          {history ? (
            <>
              <Descriptions size="small" column={3} className="dashboard-history-summary">
                <Descriptions.Item label={t("dash.historyDays")}>{t("dash.days", { n: history.days })}</Descriptions.Item>
                <Descriptions.Item label={t("dash.sessions")}>{history.total_sessions}</Descriptions.Item>
                <Descriptions.Item label={t('dash.totalLive')}>{formatDuration(history.total_seconds, en)}</Descriptions.Item>
              </Descriptions>

              <Card size="small" title={t('dash.dailyDuration')} className="dashboard-card">
                {history.daily.length === 0 ? (
                  <Empty description={t('dash.noRecord')} />
                ) : (
                  <div className="dashboard-chart dashboard-chart-small">
                    <ResponsiveContainer>
                      <BarChart data={history.daily.map((point) => ({
                        date: point.date.slice(5),
                        hours: Number((point.seconds / 3600).toFixed(2)),
                        sessions: point.sessions,
                      }))}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="date" />
                        <YAxis unit="h" />
                        <RechartsTooltip formatter={(value) => [en ? `${Number(value)}h` : `${Number(value)} 小时`, t('dash.liveDuration')]} />
                        <Bar dataKey="hours" fill="#1677ff" isAnimationActive={false} />
                      </BarChart>
                    </ResponsiveContainer>
                  </div>
                )}
              </Card>

              <Card size="small" title={t('dash.records')} className="dashboard-card">
                <Table
                  size="small"
                  rowKey="start_time"
                  dataSource={history.sessions}
                  columns={historyColumns}
                  pagination={{ pageSize: 10, hideOnSinglePage: true }}
                />
              </Card>
            </>
          ) : (
            !historyLoading && <Empty description={t('dash.noHistory')} />
          )}
        </Spin>
      </Drawer>
    </div>
  );
};

export default Dashboard;
