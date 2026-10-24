import React from 'react';
import { HashRouter as Router, Link } from 'react-router-dom';
import { Layout, Menu, Button, Tooltip } from 'antd';
import {
    MonitorOutlined,
    UnorderedListOutlined,
    DashboardOutlined,
    SettingOutlined,
    FolderOutlined,
    ToolOutlined,
    MenuFoldOutlined,
    MenuUnfoldOutlined,
    CloudUploadOutlined,
    CalendarOutlined,
    CommentOutlined,
    SyncOutlined,
    ThunderboltOutlined
} from '@ant-design/icons';
import LineChartOutlined from '@ant-design/icons/LineChartOutlined';
import { BulbOutlined, MoonOutlined, TranslationOutlined } from '@ant-design/icons';
import { useI18n } from '../../i18n';
import { useThemeMode } from '../../theme';
import './layout.css';

const { Header, Content, Sider } = Layout;
// 功能开关：IO 统计（面板明细页）
const ENABLE_IO_STATS_UI = true;

interface Props {
    children?: React.ReactNode;
}

interface InjectedProps {
    t: (key: string) => string;
    mode: 'light' | 'dark';
    toggleMode: () => void;
    toggleLocale: () => void;
    locale: 'zh' | 'en';
}

interface State {
    collapsed: boolean;
    selectedKey: string;
}

// localStorage key 用于保存侧边栏收起状态
const SIDER_COLLAPSED_KEY = 'siderCollapsed';

const ROUTE_KEYS: Record<string, string> = {
    '/dashboard': 'dashboard',
    '/': '1',
    '/liveInfo': '2',
    '/configInfo': '3',
    '/danmaku': 'danmaku',
    '/fileList': '4',
    '/tasks': 'tasks',
    '/queues': 'queues',
    '/scheduler': 'scheduler',
    '/iostats': 'iostats',
    '/update': 'update',
};

const selectedKeyFromHash = (): string => {
    const path = window.location.hash.replace(/^#/, '') || '/';
    const match = Object.keys(ROUTE_KEYS)
        .filter((route) => route !== '/' && path.startsWith(route))
        .sort((a, b) => b.length - a.length)[0];
    return match ? ROUTE_KEYS[match] : '1';
};

class RootLayoutInner extends React.Component<Props & InjectedProps, State> {
    constructor(props: Props & InjectedProps) {
        super(props);
        // 从 localStorage 读取收起状态
        let collapsed = false;
        try {
            const saved = localStorage.getItem(SIDER_COLLAPSED_KEY);
            if (saved !== null) {
                collapsed = saved === 'true';
            }
        } catch (e) {
            console.error('读取侧边栏状态失败:', e);
        }
        this.state = { collapsed, selectedKey: selectedKeyFromHash() };
    }

    componentDidMount() {
        window.addEventListener('hashchange', this.syncSelectedKey);
    }

    componentWillUnmount() {
        window.removeEventListener('hashchange', this.syncSelectedKey);
    }

    syncSelectedKey = () => {
        const selectedKey = selectedKeyFromHash();
        if (selectedKey !== this.state.selectedKey) {
            this.setState({ selectedKey });
        }
    };

    toggleCollapsed = () => {
        const collapsed = !this.state.collapsed;
        this.setState({ collapsed });
        // 保存到 localStorage
        try {
            localStorage.setItem(SIDER_COLLAPSED_KEY, String(collapsed));
        } catch (e) {
            console.error('保存侧边栏状态失败:', e);
        }
    };

    render() {
        const { collapsed } = this.state;
        const { t, mode, toggleMode, toggleLocale, locale } = this.props;
        return (
            <Router>
                <Layout className="all-layout">
                    <Header className="header small-header">
                        <h3 className="logo-text">Bililive-go</h3>
                        <div className="header-actions">
                            <Tooltip title={mode === 'dark' ? t('app.theme.light') : t('app.theme.dark')}>
                                <Button
                                    type="text"
                                    className="header-action"
                                    aria-label={mode === 'dark' ? t('app.theme.light') : t('app.theme.dark')}
                                    icon={mode === 'dark' ? <BulbOutlined /> : <MoonOutlined />}
                                    onClick={toggleMode}
                                />
                            </Tooltip>
                            <Tooltip title={t('app.locale')}>
                                <Button
                                    type="text"
                                    className="header-action header-locale"
                                    aria-label={t('app.locale')}
                                    icon={<TranslationOutlined />}
                                    onClick={toggleLocale}
                                >
                                    {locale === 'en' ? 'EN' : '中'}
                                </Button>
                            </Tooltip>
                        </div>
                    </Header>
                    <Layout>
                        <Sider
                            className="side-bar"
                            width={200}
                            collapsedWidth={60}
                            style={{ background: 'var(--surface)' }}
                            trigger={null}
                            collapsible
                            collapsed={collapsed}
                        >
                            {/* 折叠按钮在顶部，与菜单图标对齐 */}
                            <div style={{
                                padding: '12px 0',
                                borderBottom: '1px solid var(--border)',
                                width: '100%'
                            }}>
                                <Button
                                    type="text"
                                    icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                                    onClick={this.toggleCollapsed}
                                    style={{
                                        fontSize: 16,
                                        width: '100%',
                                        textAlign: 'left',
                                        paddingLeft: collapsed ? 20 : 24,
                                        height: 40
                                    }}
                                >
                                    {!collapsed && (collapsed ? t('app.expand') : t('app.collapse'))}
                                </Button>
                            </div>
                            <Menu
                                mode="inline"
                                selectedKeys={[this.state.selectedKey]}
                                inlineCollapsed={collapsed}
                                style={{ borderRight: 0 }}
                                items={[
                                    {
                                        key: 'dashboard',
                                        icon: <ThunderboltOutlined />,
                                        label: <Link to="/dashboard">{t('nav.dashboard')}</Link>,
                                    },
                                    {
                                        key: '1',
                                        icon: <MonitorOutlined />,
                                        label: <Link to="/">{t('nav.lives')}</Link>,
                                    },
                                    {
                                        key: '2',
                                        icon: <DashboardOutlined />,
                                        label: <Link to="/liveInfo">{t('nav.status')}</Link>,
                                    },
                                    {
                                        key: '3',
                                        icon: <SettingOutlined />,
                                        label: <Link to="/configInfo">{t('nav.settings')}</Link>,
                                    },
                                    {
                                        key: 'danmaku',
                                        icon: <CommentOutlined />,
                                        label: <Link to="/danmaku">{t('nav.danmaku')}</Link>,
                                    },
                                    {
                                        key: '4',
                                        icon: <FolderOutlined />,
                                        label: <Link to="/fileList">{t('nav.files')}</Link>,
                                    },
                                    {
                                        key: '5',
                                        icon: <ToolOutlined />,
                                        label: <a href="/tools/" target="_blank" rel="noopener noreferrer">{t('nav.tools')}</a>,
                                    },
                                    {
                                        key: 'tasks',
                                        icon: <UnorderedListOutlined />,
                                        label: <Link to="/tasks">{t('nav.tasks')}</Link>,
                                    },
                                    {
                                        key: 'queues',
                                        icon: <SyncOutlined />,
                                        label: <Link to="/queues">{t('nav.queues')}</Link>,
                                    },
                                    {
                                        key: 'scheduler',
                                        icon: <CalendarOutlined />,
                                        label: <a href="/scheduler/" target="_blank" rel="noopener noreferrer">{t('nav.scheduler')}</a>,
                                    },
                                    ...(ENABLE_IO_STATS_UI ? [{
                                        key: 'iostats',
                                        icon: <LineChartOutlined />,
                                        label: <Link to="/iostats">{t('nav.iostats')}</Link>,
                                    }] : []),
                                    {
                                        key: 'update',
                                        icon: <CloudUploadOutlined />,
                                        label: <Link to="/update">{t('nav.update')}</Link>,
                                    }
                                ]}
                            />
                        </Sider>
                        <Layout className="content-padding">
                            <Content
                                className="inside-content-padding"
                                style={{
                                    background: 'var(--surface)',
                                    margin: 0,
                                    minHeight: 280,
                                    overflow: "auto",
                                }}>
                                {this.props.children}
                            </Content>
                        </Layout>
                    </Layout>
                </Layout>
            </Router>
        )
    }
}

export default function RootLayout({ children }: Props) {
    const { t, toggleLocale, locale } = useI18n();
    const { mode, toggleMode } = useThemeMode();
    return (
        <RootLayoutInner t={t} mode={mode} toggleMode={toggleMode} toggleLocale={toggleLocale} locale={locale}>
            {children}
        </RootLayoutInner>
    );
}
