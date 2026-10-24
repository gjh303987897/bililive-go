import React from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';
import App from './App';
import { I18nProvider } from './i18n';
import { AppThemeProvider } from './theme';
import * as serviceWorker from './serviceWorker';

const container = document.getElementById('root');
if (!container) {
  throw new Error('Failed to find the root element');
}

// 首屏前应用持久化主题，避免明亮/暗黑闪烁
try {
  const storedTheme = localStorage.getItem('bililive.theme');
  const prefersDark = window.matchMedia?.('(prefers-color-scheme: dark)').matches;
  document.documentElement.dataset.theme = storedTheme ?? (prefersDark ? 'dark' : 'light');
  const storedLocale = localStorage.getItem('bililive.locale');
  document.documentElement.lang = storedLocale === 'en' ? 'en' : 'zh-CN';
} catch {
  // ignore
}

const root = createRoot(container);
root.render(
  <I18nProvider>
    <AppThemeProvider>
      <App />
    </AppThemeProvider>
  </I18nProvider>,
);

// If you want your app to work offline and load faster, you can change
// unregister() to register() below. Note this comes with some pitfalls.
// Learn more about service workers: https://bit.ly/CRA-PWA
serviceWorker.unregister();
