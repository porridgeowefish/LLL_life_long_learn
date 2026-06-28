import { NavLink } from 'react-router-dom';
import clsx from 'clsx';

import { ConnectionBadge } from './ConnectionBadge';
import s from './Topbar.module.css';

interface NavItem {
  to: string;
  label: string;
  exact?: boolean;
}

const NAV_ITEMS: readonly NavItem[] = [
  { to: '/', label: '总览', exact: true },
  { to: '/project', label: '项目' },
  { to: '/agents', label: '智能体' },
  { to: '/memory', label: '记忆' },
  { to: '/settings', label: '配置' },
];

export function Topbar() {
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
        <ConnectionBadge />
      </div>
    </header>
  );
}
