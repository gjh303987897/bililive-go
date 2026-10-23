import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  Empty,
  Popconfirm,
  Progress,
  Space,
  Spin,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import {
  CheckCircleOutlined,
  CloudUploadOutlined,
  DeleteOutlined,
  ReloadOutlined,
  SyncOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import './index.css';

const { Text } = Typography;

type QueueStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

interface QueueFile {
  path: string;
  type: string;
  size: number;
  deletable?: boolean;
}

interface QueueRecord {
  live_id: string;
  platform: string;
  host_name: string;
  room_name: string;
  start_time: string;
}

interface QueueTask {
  id: number;
  status: QueueStatus;
  files: QueueFile[];
  record_info: QueueRecord;
  error_message?: string;
  progress: number;
  created_at: string;
  can_retry: boolean;
}

interface QueueData {
  transcode_waiting: QueueTask[];
  transcode_failed: QueueTask[];
  upload_pending: QueueTask[];
  upload_failed: QueueTask[];
  upload_completed: QueueTask[];
  upload_enabled: boolean;
}

const STATUS_TAGS: Record<QueueStatus, { color: string; label: string }> = {
  pending: { color: 'default', label: '等待中' },
  running: { color: 'processing', label: '执行中' },
  completed: { color: 'success', label: '已完成' },
  failed: { color: 'error', label: '失败' },
  cancelled: { color: 'default', label: '已取消' },
};

const fileNames = (task: QueueTask): string =>
  task.files.map((file) => file.path.split(/[\\/]/).pop() || file.path).join('、');

const formatTime = (value: string): string => (value ? dayjs(value).format('MM-DD HH:mm:ss') : '-');

const Queues: React.FC = () => {
  const [data, setData] = useState<QueueData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [acting, setActing] = useState<number | null>(null);

  const load = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const response = await fetch('/api/pipeline/queues');
      const payload = await response.json();
      if (!response.ok || payload.err_no) {
        throw new Error(payload.err_msg || `HTTP ${response.status}`);
      }
      setData(payload.data as QueueData);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      if (!silent) setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
    const timer = window.setInterval(() => {
      if (!document.hidden) load(true);
    }, 5000);
    return () => window.clearInterval(timer);
  }, [load]);

  const act = useCallback(async (task: QueueTask, action: 'retry' | 'cancel' | 'delete') => {
    setActing(task.id);
    try {
      const method = action === 'delete' ? 'DELETE' : 'POST';
      const url = action === 'delete'
        ? `/api/pipeline/tasks/${task.id}`
        : `/api/pipeline/tasks/${task.id}/${action}`;
      const response = await fetch(url, { method });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok || payload.err_no) {
        throw new Error(payload.err_msg || `HTTP ${response.status}`);
      }
      await load(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setActing(null);
    }
  }, [load]);

  const clearFailed = useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/pipeline/queues/transcode/clear-failed', { method: 'POST' });
      const payload = await response.json();
      if (!response.ok || payload.err_no) {
        throw new Error(payload.err_msg || `HTTP ${response.status}`);
      }
      await load(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [load]);

  const baseColumns = useMemo<ColumnsType<QueueTask>>(() => [
    {
      title: '文件',
      key: 'files',
      render: (_, task) => (
        <Tooltip title={task.files.map((file) => file.path).join('\n')}>
          <span>{fileNames(task) || '-'}</span>
        </Tooltip>
      ),
    },
    {
      title: '直播间',
      key: 'room',
      width: 180,
      render: (_, task) => (
        <span>{task.record_info.host_name || task.record_info.room_name || task.record_info.live_id}</span>
      ),
    },
    {
      title: '状态',
      key: 'status',
      width: 150,
      render: (_, task) => {
        const tag = STATUS_TAGS[task.status] ?? STATUS_TAGS.pending;
        return (
          <Space direction="vertical" size={0}>
            <Tag color={tag.color}>{tag.label}</Tag>
            {task.status === 'running' && <Progress percent={task.progress} size="small" />}
          </Space>
        );
      },
    },
    {
      title: '创建时间',
      key: 'created_at',
      width: 130,
      render: (_, task) => formatTime(task.created_at),
    },
  ], []);

  const waitingColumns = useMemo<ColumnsType<QueueTask>>(() => [
    ...baseColumns,
    {
      title: '操作',
      key: 'actions',
      width: 120,
      render: (_, task) => (
        <Space>
          {task.status === 'running' && (
            <Button size="small" loading={acting === task.id} onClick={() => act(task, 'cancel')}>取消</Button>
          )}
          <Popconfirm title="删除任务？本地文件不会删除" onConfirm={() => act(task, 'delete')}>
            <Button size="small" icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ], [act, acting, baseColumns]);

  const failedColumns = useMemo<ColumnsType<QueueTask>>(() => [
    ...baseColumns,
    {
      title: '错误',
      key: 'error',
      render: (_, task) => (
        task.status === 'failed'
          ? <Text type="danger" ellipsis={{ tooltip: task.error_message }}>{task.error_message || '未知错误'}</Text>
          : '-'
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 160,
      render: (_, task) => (
        <Space>
          {task.status === 'failed' && (
            <Button size="small" type="primary" loading={acting === task.id} onClick={() => act(task, 'retry')}>
              重试
            </Button>
          )}
          {task.status === 'running' && (
            <Button size="small" loading={acting === task.id} onClick={() => act(task, 'cancel')}>取消</Button>
          )}
          <Popconfirm title="从队列移除？本地文件不会删除" onConfirm={() => act(task, 'delete')}>
            <Button size="small" icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ], [act, acting, baseColumns]);

  const completedColumns = useMemo<ColumnsType<QueueTask>>(() => [
    ...baseColumns,
    {
      title: '远程路径',
      key: 'remote',
      render: (_, task) => {
        const remote = task.files.find((file) => (file as QueueFile & { metadata?: Record<string, string> }).metadata?.remote_path);
        if (!remote) return '-';
        const value = (remote as QueueFile & { metadata?: Record<string, string> }).metadata?.remote_path;
        return <Text ellipsis={{ tooltip: value }}>{value}</Text>;
      },
    },
  ], [baseColumns]);

  const uploadQueue = useMemo(
    () => [...(data?.upload_pending ?? []), ...(data?.upload_failed ?? [])],
    [data],
  );

  return (
    <div className="queues-page">
      <Card size="small" className="queues-toolbar">
        <Space wrap>
          <Button icon={<ReloadOutlined />} loading={loading} onClick={() => load()}>刷新</Button>
          <Text type="secondary">自动刷新 5 秒</Text>
          {data && (
            <Space size="large">
              <Badge status="processing" text={`等待转码 ${data.transcode_waiting.length}`} />
              <Badge status="error" text={`转码失败 ${data.transcode_failed.length}`} />
              <Badge status="warning" text={`上传 ${uploadQueue.length}`} />
            </Space>
          )}
        </Space>
      </Card>

      {error && (
        <Alert type="error" showIcon closable message="队列加载失败" description={error} onClose={() => setError(null)} className="queues-alert" />
      )}

      {data && !data.upload_enabled && (
        <Alert
          type="info"
          showIcon
          className="queues-alert"
          message="未启用远程上传"
          description="在配置文件 remote_upload 中启用 WebDAV 或 S3 后，转码完成的视频会自动进入上传队列。"
        />
      )}

      <Spin spinning={loading}>
        <Tabs
          items={[
            {
              key: 'transcode-waiting',
              label: <span><SyncOutlined />等待转码</span>,
              children: (
                <Card size="small">
                  <Table
                    size="small"
                    rowKey="id"
                    dataSource={data?.transcode_waiting ?? []}
                    columns={waitingColumns}
                    pagination={{ pageSize: 20, hideOnSinglePage: true }}
                    locale={{ emptyText: <Empty description="没有等待转码的任务" /> }}
                  />
                </Card>
              ),
            },
            {
              key: 'transcode-failed',
              label: <span><WarningOutlined />转码失败</span>,
              children: (
                <Card
                  size="small"
                  title="失败的任务会保留源文件，可重试或手动处理"
                  extra={(
                    <Popconfirm title="清空失败队列？本地文件不会删除" onConfirm={clearFailed}>
                      <Button size="small" danger>清空失败队列</Button>
                    </Popconfirm>
                  )}
                >
                  <Table
                    size="small"
                    rowKey="id"
                    dataSource={data?.transcode_failed ?? []}
                    columns={failedColumns}
                    pagination={{ pageSize: 20, hideOnSinglePage: true }}
                    locale={{ emptyText: <Empty description="没有转码失败的任务" /> }}
                  />
                </Card>
              ),
            },
            {
              key: 'upload',
              label: <span><CloudUploadOutlined />上传队列</span>,
              children: (
                <>
                  <Card size="small" title="待上传 / 上传失败" className="queues-card">
                    <Table
                      size="small"
                      rowKey="id"
                      dataSource={uploadQueue}
                      columns={failedColumns}
                      pagination={{ pageSize: 20, hideOnSinglePage: true }}
                      locale={{ emptyText: <Empty description={data?.upload_enabled ? '没有待上传任务' : '远程上传未启用'} /> }}
                    />
                  </Card>
                  <Card size="small" title={<span><CheckCircleOutlined /> 最近上传完成</span>}>
                    <Table
                      size="small"
                      rowKey="id"
                      dataSource={(data?.upload_completed ?? []).slice(0, 20)}
                      columns={completedColumns}
                      pagination={false}
                      locale={{ emptyText: <Empty description="没有已完成的上传" /> }}
                    />
                  </Card>
                </>
              ),
            },
          ]}
        />
      </Spin>
    </div>
  );
};

export default Queues;
