import { useState } from 'react';
import { NavLink } from 'react-router-dom';
import clsx from 'clsx';

import { requestShutdown } from '@/api/system';
import { Button } from '@/components/primitive/Button';
import { Icon } from '@/components/primitive/Icon';

import { ConnectionBadge } from './ConnectionBadge';
import s from './Topbar.module.css';

interface NavItem {
  to: string;
  label: string;
  exact?: boolean;
}

const NAV_ITEMS: readonly NavItem[] = [
  { to: '/', label: '主页', exact: true },
  { to: '/agents', label: '智能体' },
  { to: '/memory', label: '记忆' },
  { to: '/settings', label: '配置' },
];

export function Topbar() {
  const [exiting, setExiting] = useState(false);

  const handleExit = async () => {
    if (exiting) return;
    const ok = window.confirm('关闭 LifeLongLearn 本地服务？');
    if (!ok) return;
    setExiting(true);
    try {
      await requestShutdown();
    } catch (err) {
      setExiting(false);
      window.alert(`关闭失败：${(err as Error).message}`);
    }
  };

  return (
    <header className={s.topbar}>
      <a href="/" className={s.logo}>
        <img className={s.logoMark} src="/logo-lychee.svg" alt="" aria-hidden="true" />
        <span>Life<span className={s.accent}>Long</span>Learn</span>
      </a>
      <span className={s.sep} />
      <nav className={s.nav}>
        {NAV_ITEMS.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.exact}
            className={({ isActive }) => clsx(s.link, isActive && s.active)}
          >
            {item.label}
          </NavLink>
        ))}
      </nav>
      <div className={s.right}>
        <a
          className={s.github}
          href="https://github.com/porridgeowefish/LLL_life_long_learn"
          target="_blank"
          rel="noreferrer"
          aria-label="在 GitHub 查看 LifeLongLearn 源码，可提交 Issue 和 Pull Request"
        >
          GitHub
        </a>
        <ConnectionBadge />
        <Button
          size="sm"
          variant="ghost"
          className={s.exitBtn}
          iconLeft={<Icon name="x" size={13} />}
          loading={exiting}
          onClick={handleExit}
          title="关闭本地服务"
        >
          退出
        </Button>
      </div>
    </header>
  );
}
