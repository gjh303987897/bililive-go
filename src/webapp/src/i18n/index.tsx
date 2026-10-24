import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';

export type Locale = 'zh' | 'en';

type Dictionary = Record<string, string>;

const STORAGE_KEY = 'bililive.locale';

const zh: Dictionary = {
  'app.collapse': '收起菜单',
  'app.expand': '展开菜单',
  'app.theme.dark': '切换暗黑模式',
  'app.theme.light': '切换明亮模式',
  'app.locale': '切换语言',
  'nav.dashboard': '监控面板',
  'nav.lives': '监控列表',
  'nav.status': '系统状态',
  'nav.settings': '设置',
  'nav.danmaku': '弹幕',
  'nav.files': '文件',
  'nav.tools': '工具',
  'nav.tasks': '任务队列',
  'nav.queues': '转码/上传队列',
  'nav.scheduler': '调度器',
  'nav.iostats': 'IO 统计',
  'nav.update': '更新',

  'common.refresh': '刷新',
  'common.retry': '重试',
  'common.cancel': '取消',
  'common.delete': '删除',
  'common.clear': '清空',
  'common.close': '关闭',
  'common.loading': '加载中',
  'common.empty': '暂无数据',
  'common.updatedAt': '更新于',

  'queue.title': '队列处理',
  'queue.autoRefresh': '每 {n} 秒自动刷新',
  'queue.waiting': '等待转码',
  'queue.failed': '转码失败',
  'queue.uploadPending': '待上传',
  'queue.completed': '最近完成',
  'queue.tabUpload': '上传队列',
  'queue.file': '文件',
  'queue.room': '直播间',
  'queue.status': '状态',
  'queue.created': '创建时间',
  'queue.actions': '操作',
  'queue.error': '错误',
  'queue.remotePath': '远程路径',
  'queue.running': '执行中',
  'queue.pending': '等待中',
  'queue.done': '已完成',
  'queue.cancelled': '已取消',
  'queue.noWaiting': '没有等待转码的任务',
  'queue.noFailed': '没有转码失败的任务',
  'queue.noUpload': '没有待上传任务',
  'queue.noCompleted': '没有已完成的上传',
  'queue.failedHint': '失败的任务会保留源文件，可重试或手动处理',
  'queue.clearFailed': '清空失败队列',
  'queue.clearFailedConfirm': '清空失败队列？本地文件不会删除',
  'queue.removeConfirm': '从队列移除？本地文件不会删除',
  'queue.pendingUploadTitle': '待上传 / 上传失败',
  'queue.recentCompleted': '最近上传完成',
  'queue.loadFailed': '队列加载失败',
  'queue.uploadDisabled': '未启用远程上传',
  'queue.uploadDisabledHint': '在配置文件 remote_upload 中启用 WebDAV 或 S3 后，转码完成的视频会自动进入上传队列。',
  'queue.sessions': '{n} 个直播会话',
  'queue.unknownError': '未知错误',

  'dash.title': '监控面板',
  'dash.window': '统计窗口',
  'dash.refresh': '自动刷新',
  'dash.range1h': '最近 1 小时',
  'dash.range6h': '最近 6 小时',
  'dash.range24h': '最近 24 小时',
  'dash.range7d': '最近 7 天',
  'dash.range30d': '最近 30 天',
  'dash.off': '关闭',
  'dash.rawIO': '原始 IO 数据',
  'dash.overview': '实时概览',
  'dash.system': '系统资源',
  'dash.rooms': '直播间',
  'dash.monitoring': '监控中',
  'dash.liveRecording': '直播中 / 录制中',
  'dash.totalBytes': '录制文件总量',
  'dash.bitrate': '下载速率',
  'dash.bitrateEmpty': '当前窗口没有速率数据',
  'dash.ioDisabled': 'IO 统计模块未启用',
  'dash.rank': '窗口内开播时长排行',
  'dash.rankEmpty': '当前窗口没有开播记录',
  'dash.daily': '每日开播时长（窗口内）',
  'dash.details': '直播间明细',
  'dash.platform': '平台',
  'dash.hostRoom': '主播 / 房间',
  'dash.liveDuration': '开播时长',
  'dash.recordDuration': '本次录制',
  'dash.fileSize': '文件大小',
  'dash.speed': '当前速率',
  'dash.inWindow': '窗口内开播',
  'dash.openRoom': '监控页',
  'dash.history': '开播历史',
  'dash.historyOf': '{name} 的开播历史',
  'dash.historyDays': '窗口',
  'dash.days': '{n} 天',
  'dash.sessions': '开播场次',
  'dash.totalLive': '累计开播',
  'dash.dailyDuration': '每日开播时长',
  'dash.records': '开播记录',
  'dash.startAt': '开播时间',
  'dash.endAt': '下播时间',
  'dash.duration': '时长',
  'dash.endReason': '结束原因',
  'dash.liveNow': '直播中',
  'dash.noRecord': '窗口内没有开播记录',
  'dash.noHistory': '暂无记录',
  'dash.loadFailed': '监控数据加载失败',
  'dash.loading': '加载中',
  'dash.recording': '录制中',
  'dash.preparing': '录制准备中',
  'dash.live': '直播中',
  'dash.listening': '监控中',
  'dash.idle': '已停止',
  'dash.sessionsTooltip': '{n} 场',
  'dash.reason.normal': '正常下播',
  'dash.reason.user_stop': '停止监控',
  'dash.reason.crash': '程序崩溃',
  'dash.reason.error': '录制异常',
  'dash.reason.unknown': '未知',
};

const en: Dictionary = {
  'app.collapse': 'Collapse menu',
  'app.expand': 'Expand menu',
  'app.theme.dark': 'Switch to dark mode',
  'app.theme.light': 'Switch to light mode',
  'app.locale': 'Switch language',
  'nav.dashboard': 'Dashboard',
  'nav.lives': 'Monitored rooms',
  'nav.status': 'System status',
  'nav.settings': 'Settings',
  'nav.danmaku': 'Danmaku',
  'nav.files': 'Files',
  'nav.tools': 'Tools',
  'nav.tasks': 'Task queue',
  'nav.queues': 'Transcode / upload',
  'nav.scheduler': 'Scheduler',
  'nav.iostats': 'IO stats',
  'nav.update': 'Update',

  'common.refresh': 'Refresh',
  'common.retry': 'Retry',
  'common.cancel': 'Cancel',
  'common.delete': 'Delete',
  'common.clear': 'Clear',
  'common.close': 'Close',
  'common.loading': 'Loading',
  'common.empty': 'No data',
  'common.updatedAt': 'Updated at',

  'queue.title': 'Queue processing',
  'queue.autoRefresh': 'Auto refresh every {n}s',
  'queue.waiting': 'Waiting',
  'queue.failed': 'Transcode failures',
  'queue.uploadPending': 'Pending uploads',
  'queue.completed': 'Recently completed',
  'queue.tabUpload': 'Upload queue',
  'queue.file': 'File',
  'queue.room': 'Room',
  'queue.status': 'Status',
  'queue.created': 'Created',
  'queue.actions': 'Actions',
  'queue.error': 'Error',
  'queue.remotePath': 'Remote path',
  'queue.running': 'Running',
  'queue.pending': 'Pending',
  'queue.done': 'Completed',
  'queue.cancelled': 'Cancelled',
  'queue.noWaiting': 'No files waiting to transcode',
  'queue.noFailed': 'No transcode failures',
  'queue.noUpload': 'No pending uploads',
  'queue.noCompleted': 'No completed uploads',
  'queue.failedHint': 'Failed tasks keep their source files and can be retried',
  'queue.clearFailed': 'Clear failures',
  'queue.clearFailedConfirm': 'Clear failed queue? Local files are kept',
  'queue.removeConfirm': 'Remove from queue? Local files are kept',
  'queue.pendingUploadTitle': 'Pending / failed uploads',
  'queue.recentCompleted': 'Recently completed',
  'queue.loadFailed': 'Failed to load queues',
  'queue.uploadDisabled': 'Remote upload is disabled',
  'queue.uploadDisabledHint': 'Enable WebDAV or S3 under remote_upload to enqueue transcoded videos automatically.',
  'queue.sessions': '{n} sessions',
  'queue.unknownError': 'Unknown error',

  'dash.title': 'Dashboard',
  'dash.window': 'Window',
  'dash.refresh': 'Auto refresh',
  'dash.range1h': 'Last hour',
  'dash.range6h': 'Last 6 hours',
  'dash.range24h': 'Last 24 hours',
  'dash.range7d': 'Last 7 days',
  'dash.range30d': 'Last 30 days',
  'dash.off': 'Off',
  'dash.rawIO': 'Raw IO data',
  'dash.overview': 'Overview',
  'dash.system': 'System',
  'dash.rooms': 'Rooms',
  'dash.monitoring': 'Monitoring',
  'dash.liveRecording': 'Live / recording',
  'dash.totalBytes': 'Recorded size',
  'dash.bitrate': 'Download bitrate',
  'dash.bitrateEmpty': 'No bitrate data in this window',
  'dash.ioDisabled': 'IO stats module is disabled',
  'dash.rank': 'Live duration ranking',
  'dash.rankEmpty': 'No live sessions in this window',
  'dash.daily': 'Daily live duration (window)',
  'dash.details': 'Room details',
  'dash.platform': 'Platform',
  'dash.hostRoom': 'Host / room',
  'dash.liveDuration': 'Live for',
  'dash.recordDuration': 'Recording for',
  'dash.fileSize': 'File size',
  'dash.speed': 'Current rate',
  'dash.inWindow': 'Live in window',
  'dash.openRoom': 'Monitor page',
  'dash.history': 'Live history',
  'dash.historyOf': 'Live history of {name}',
  'dash.historyDays': 'Window',
  'dash.days': '{n} days',
  'dash.sessions': 'Sessions',
  'dash.totalLive': 'Total live',
  'dash.dailyDuration': 'Daily live duration',
  'dash.records': 'Sessions',
  'dash.startAt': 'Started',
  'dash.endAt': 'Ended',
  'dash.duration': 'Duration',
  'dash.endReason': 'End reason',
  'dash.liveNow': 'Live',
  'dash.noRecord': 'No sessions in this window',
  'dash.noHistory': 'No records',
  'dash.loadFailed': 'Failed to load dashboard',
  'dash.loading': 'Loading',
  'dash.recording': 'Recording',
  'dash.preparing': 'Preparing',
  'dash.live': 'Live',
  'dash.listening': 'Monitoring',
  'dash.idle': 'Stopped',
  'dash.sessionsTooltip': '{n} sessions',
  'dash.reason.normal': 'Ended normally',
  'dash.reason.user_stop': 'Monitoring stopped',
  'dash.reason.crash': 'App crash',
  'dash.reason.error': 'Recording error',
  'dash.reason.unknown': 'Unknown',
};

const DICTIONARIES: Record<Locale, Dictionary> = { zh, en };

interface I18nContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  toggleLocale: () => void;
  t: (key: string, vars?: Record<string, string | number>) => string;
}

const I18nContext = createContext<I18nContextValue | null>(null);

const readStoredLocale = (): Locale => {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'zh' || stored === 'en') return stored;
  } catch {
    // ignore
  }
  return navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en';
};

export const I18nProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [locale, setLocaleState] = useState<Locale>(readStoredLocale);

  useEffect(() => {
    document.documentElement.lang = locale === 'zh' ? 'zh-CN' : 'en';
  }, [locale]);

  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next);
    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // ignore
    }
    document.documentElement.lang = next === 'zh' ? 'zh-CN' : 'en';
  }, []);

  const toggleLocale = useCallback(() => {
    setLocale(locale === 'zh' ? 'en' : 'zh');
  }, [locale, setLocale]);

  const t = useCallback((key: string, vars?: Record<string, string | number>) => {
    const template = DICTIONARIES[locale][key] ?? DICTIONARIES.zh[key] ?? key;
    if (!vars) return template;
    return template.replace(/\{(\w+)\}/g, (match, name: string) =>
      (vars[name] !== undefined ? String(vars[name]) : match));
  }, [locale]);

  const value = useMemo(
    () => ({ locale, setLocale, toggleLocale, t }),
    [locale, setLocale, toggleLocale, t],
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
};

export const useI18n = (): I18nContextValue => {
  const context = useContext(I18nContext);
  if (!context) throw new Error('useI18n must be used inside I18nProvider');
  return context;
};
