import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Alert,
  Button,
  Card,
  Col,
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
}

const RANGE_OPTIONS = [
  { value: 60 * 60 * 1000, label: '最近 1 小时' },
  { value: 6 * 60 * 60 * 1000, label: '最近 6 小时' },
  { value: 24 * 60 * 60 * 1000, label: '最近 24 小时' },
  { value: 7 * 24 * 60 * 60 * 1000, label: '最近 7 天' },
  { value: 30 * 24 * 60 * 60 * 1000, label: '最近 30 天' },
];

const REFRESH_OPTIONS = [
  { value: 5000, label: '5 秒' },
  { value: 10000, label: '10 秒' },
  { value: 30000, label: '30 秒' },
  { value: 60000, label: '1 分钟' },
  { value: 0, label: '关闭' },
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

const formatDuration = (seconds: number): string => {
  if (!seconds || seconds <= 0) return '-';
  const total = Math.floor(seconds);
  const days = Math.floor(total / 86400);
  const hours = Math.floor((total % 86400) / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  if (days > 0) return `${days} 天 ${hours} 小时`;
  if (hours > 0) return `${hours} 小时 ${minutes} 分`;
  if (minutes > 0) return `${minutes} 分 ${secs} 秒`;
  return `${secs} 秒`;
};

const roomLabel = (room: Pick<DashboardRoom, 'host_name' | 'room_name' | 'live_id'>): string =>
  room.host_name || room.room_name || room.live_id;

const Dashboard: React.FC = () => {
  const [range, setRange] = useState(RANGE_OPTIONS[2].value);
  const [refreshInterval, setRefreshInterval] = useState(10000);
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [updatedAt, setUpdatedAt] = useState<number>(0);
  const inFlight = useRef(false);

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

  const columns = useMemo<ColumnsType<DashboardRoom>>(() => [
    {
      title: '状态',
      key: 'status',
      width: 130,
      render: (_, room) => {
        const tags: React.ReactNode[] = [];
        if (room.recording) tags.push(<Tag color="red" key="rec">录制中</Tag>);
        if (room.recording_preparing) tags.push(<Tag color="orange" key="prep">录制准备中</Tag>);
        if (room.status) tags.push(<Tag color="green" key="live">直播中</Tag>);
        if (room.listening) tags.push(<Tag color="blue" key="listen">监控中</Tag>);
        if (tags.length === 0) tags.push(<Tag key="idle">已停止</Tag>);
        return <Space size={4} wrap>{tags}</Space>;
      },
    },
    {
      title: '平台',
      dataIndex: 'platform',
      key: 'platform',
      width: 90,
    },
    {
      title: '主播 / 房间',
      key: 'room',
      render: (_, room) => (
        <div>
          <Link to={`/?room=${room.live_id}`}>{roomLabel(room)}</Link>
          <div className="dashboard-room-sub">{room.room_name}</div>
        </div>
      ),
    },
    {
      title: '开播时长',
      dataIndex: 'live_seconds',
      key: 'live_seconds',
      width: 110,
      sorter: (a, b) => a.live_seconds - b.live_seconds,
      render: (value: number) => formatDuration(value),
    },
    {
      title: '本次录制',
      dataIndex: 'recorded_seconds',
      key: 'recorded_seconds',
      width: 110,
      sorter: (a, b) => a.recorded_seconds - b.recorded_seconds,
      render: (value: number) => formatDuration(value),
    },
    {
      title: '文件大小',
      dataIndex: 'total_bytes',
      key: 'total_bytes',
      width: 110,
      sorter: (a, b) => a.total_bytes - b.total_bytes,
      render: (value: number) => formatBytes(value),
    },
    {
      title: '当前速率',
      dataIndex: 'speed',
      key: 'speed',
      width: 110,
      sorter: (a, b) => a.speed - b.speed,
      render: (value: number) => formatSpeed(value),
    },
    {
      title: '窗口内开播',
      dataIndex: 'seconds_in_range',
      key: 'seconds_in_range',
      width: 130,
      sorter: (a, b) => a.seconds_in_range - b.seconds_in_range,
      render: (value: number, room) => (
        <Tooltip title={`${room.sessions_in_range} 个直播会话`}>{formatDuration(value)}</Tooltip>
      ),
    },
  ], []);

  const summary = data?.summary;

  return (
    <div className="dashboard-page">
      <Card size="small" className="dashboard-toolbar">
        <Space wrap>
          <Text strong>统计窗口</Text>
          <Select
            value={range}
            style={{ width: 140 }}
            options={RANGE_OPTIONS}
            onChange={setRange}
          />
          <Text strong>自动刷新</Text>
          <Select
            value={refreshInterval}
            style={{ width: 110 }}
            options={REFRESH_OPTIONS}
            onChange={setRefreshInterval}
          />
          <Button icon={<ReloadOutlined />} loading={loading} onClick={load}>刷新</Button>
          <Link to="/iostats">
            <Button icon={<DatabaseOutlined />} type="link">原始 IO 数据</Button>
          </Link>
          <Text type="secondary">
            更新于 {updatedAt ? dayjs(updatedAt).format('HH:mm:ss') : '-'}
          </Text>
        </Space>
      </Card>

      {error && (
        <Alert
          type="error"
          showIcon
          closable
          message="监控数据加载失败"
          description={error}
          onClose={() => setError(null)}
          className="dashboard-alert"
        />
      )}

      <Spin spinning={loading}>
        <Row gutter={16} className="dashboard-summary">
          <Col xs={12} md={6}>
            <Card size="small">
              <Statistic title="直播间" value={summary?.rooms ?? 0} prefix={<VideoCameraOutlined />} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size="small">
              <Statistic title="监控中" value={summary?.listening ?? 0} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size="small">
              <Statistic title="直播中 / 录制中" value={summary?.live ?? 0} suffix={`/ ${summary?.recording ?? 0}`} />
            </Card>
          </Col>
          <Col xs={12} md={6}>
            <Card size="small">
              <Statistic title="录制文件总量" value={formatBytes(summary?.total_bytes ?? 0)} />
            </Card>
          </Col>
        </Row>

        <Tabs
          defaultActiveKey="overview"
          items={[
            {
              key: 'overview',
              label: <span><ThunderboltOutlined />实时概览</span>,
              children: (
                <>
                  <Card size="small" title="下载速率" className="dashboard-card">
                    {bitrateChart.rows.length === 0 ? (
                      <Empty
                        description={data?.iostats_enabled === false ? 'IO 统计模块未启用' : '当前窗口没有速率数据'}
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

                  <Card size="small" title="窗口内开播时长排行" className="dashboard-card">
                    {rankingChart.length === 0 ? (
                      <Empty description="当前窗口没有开播记录" />
                    ) : (
                      <div className="dashboard-chart dashboard-chart-small">
                        <ResponsiveContainer>
                          <BarChart data={rankingChart} layout="vertical" margin={{ left: 24 }}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis type="number" unit="h" />
                            <YAxis type="category" dataKey="name" width={120} />
                            <RechartsTooltip formatter={(value) => [`${Number(value)} 小时`, '开播时长']} />
                            <Bar dataKey="hours" fill="#1677ff" isAnimationActive={false} />
                          </BarChart>
                        </ResponsiveContainer>
                      </div>
                    )}
                  </Card>

                  <Card size="small" title="直播间明细" className="dashboard-card">
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
              label: <span><AreaChartOutlined />系统资源</span>,
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
    </div>
  );
};

export default Dashboard;
