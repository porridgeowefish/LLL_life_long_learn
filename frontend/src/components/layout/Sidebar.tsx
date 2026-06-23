import { useLayoutEffect, useRef, useState, type CSSProperties } from 'react';
import { Link } from 'react-router-dom';
import { useUiStore } from '@/store';
import { ChevronDownIcon, PlusIcon } from '@radix-ui/react-icons';

import { useProjects } from '@/api/projects';
import { useFolders, useSaveFolders } from '@/api/folders';
import {
  assignProject,
  createFolder,
  deleteFolder,
  folderOf,
  renameFolder,
  uncategorizedSlugs,
  type FolderLayout,
} from '@/lib/folders';
import { Icon, type IconName } from '@/components/primitive/Icon';

import s from './Sidebar.module.css';

const QUICK_LINKS: ReadonlyArray<{ to: string; label: string; icon: IconName }> = [
  { to: '/', label: '学习总览', icon: 'zap' },
  { to: '/agents', label: '智能体管理', icon: 'bot' },
  { to: '/memory', label: '记忆系统', icon: 'brain' },
];

export function Sidebar() {
  const { data: projectsData, isLoading } = useProjects();
  const projects = projectsData?.projects ?? [];
  const { data: foldersData } = useFolders();
  const layout: FolderLayout = foldersData ?? { folders: [] };
  const save = useSaveFolders();

  const allSlugs = projects.map((p) => p.slug);
  const uncategorized = uncategorizedSlugs(layout, allSlugs);
  const projectBySlug = new Map(projects.map((p) => [p.slug, p] as const));

  // Collapsed folders are a persisted UI pref (ui Zustand slice → localStorage),
  // so expand/collapse choices survive reload. Default: all expanded.
  const collapsedFolderIds = useUiStore((st) => st.collapsedFolderIds);
  const toggleFolderCollapsed = useUiStore((st) => st.toggleFolderCollapsed);
  const collapsed = new Set(collapsedFolderIds);

  const [creating, setCreating] = useState(false);
  const [draftName, setDraftName] = useState('');
  const [renamingId, setRenamingId] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState('');
  const [menuSlug, setMenuSlug] = useState<string | null>(null);

  const commit = (next: FolderLayout) => save.mutate(next);

  const submitCreate = () => {
    const name = draftName.trim();
    if (name) commit(createFolder(layout, name));
    setDraftName('');
    setCreating(false);
  };

  const submitRename = () => {
    if (renamingId) commit(renameFolder(layout, renamingId, renameValue));
    setRenamingId(null);
    setRenameValue('');
  };

  const removeFolder = (id: string, name: string) => {
    if (window.confirm(`删除文件夹「${name}」？其中的项目会回到未分类。`)) {
      commit(deleteFolder(layout, id));
    }
  };

  const moveProject = (slug: string, target: string | null) => {
    commit(assignProject(layout, slug, target));
  };

  const renderProject = (slug: string) => {
    const p = projectBySlug.get(slug);
    if (!p) return null;
    const currentFolderId = folderOf(layout, p.slug);
    return (
      <ProjectRow
        key={p.slug}
        slug={p.slug}
        title={p.title}
        currentFolderId={currentFolderId}
        folders={layout.folders}
        menuOpen={menuSlug === p.slug}
        onToggleMenu={() => setMenuSlug(menuSlug === p.slug ? null : p.slug)}
        onCloseMenu={() => setMenuSlug(null)}
        onMove={(target) => moveProject(p.slug, target)}
      />
    );
  };

  return (
    <aside className={s.sidebar}>
      <div className={s.head}>
        <span>导航</span>
      </div>
      <div className={s.scroll}>
        <div className={s.section}>快捷入口</div>
        {QUICK_LINKS.map((q) => (
          <Link key={q.to} to={q.to} className={s.link}>
            <Icon name={q.icon} size={15} className={s.linkIcon} />
            {q.label}
          </Link>
        ))}

        <div className={s.sectionRow}>
          <div className={s.section}>项目</div>
          <button
            type="button"
            className={s.iconBtn}
            title="新建文件夹"
            onClick={() => setCreating(true)}
          >
            <PlusIcon />
          </button>
        </div>

        {save.isPending && <div className={s.saving}>保存中…</div>}
        {save.isError && (
          <div className={s.saveError} role="alert">
            文件夹保存失败：{(save.error as Error).message}（请确认后端已重启）
          </div>
        )}

        {isLoading && <div className={s.muted}>加载中…</div>}
        {!isLoading && projects.length === 0 && (
          <div className={s.muted}>尚无项目，去总览创建一个 →</div>
        )}

        {layout.folders.map((f) => {
          const isOpen = !collapsed.has(f.id);
          const isRenaming = renamingId === f.id;
          return (
            <div key={f.id} className={s.folder}>
              <div className={s.folderHeader}>
                <button
                  type="button"
                  className={s.chevronBtn}
                  onClick={() => toggleFolderCollapsed(f.id)}
                  aria-label={isOpen ? '收起' : '展开'}
                >
                  <ChevronDownIcon className={isOpen ? undefined : s.chevronClosed} />
                </button>
                {isRenaming ? (
                  <input
                    className={s.folderInput}
                    autoFocus
                    value={renameValue}
                    onChange={(e) => setRenameValue(e.target.value)}
                    onBlur={submitRename}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') submitRename();
                      if (e.key === 'Escape') {
                        setRenamingId(null);
                        setRenameValue('');
                      }
                    }}
                  />
                ) : (
                  <button
                    type="button"
                    className={s.folderName}
                    title="单击收起/展开 · 双击重命名"
                    onClick={() => toggleFolderCollapsed(f.id)}
                    onDoubleClick={() => {
                      setRenamingId(f.id);
                      setRenameValue(f.name);
                    }}
                  >
                    <span className={s.folderNameText}>{f.name}</span>
                    <span className={s.count}>{f.slugOrder.length}</span>
                  </button>
                )}
                <button
                  type="button"
                  className={s.iconBtn}
                  title="删除文件夹"
                  onClick={() => removeFolder(f.id, f.name)}
                >
                  <Icon name="x" size={13} />
                </button>
              </div>
              {isOpen && <div className={s.folderBody}>{f.slugOrder.map(renderProject)}</div>}
            </div>
          );
        })}

        {uncategorized.length > 0 && layout.folders.length > 0 && (
          <div className={s.subSection}>未分类</div>
        )}
        {uncategorized.map(renderProject)}

        {creating && (
          <input
            className={`${s.folderInput} ${s.createInput}`}
            autoFocus
            placeholder="文件夹名称，回车创建"
            value={draftName}
            onChange={(e) => setDraftName(e.target.value)}
            onBlur={submitCreate}
            onKeyDown={(e) => {
              if (e.key === 'Enter') submitCreate();
              if (e.key === 'Escape') {
                setCreating(false);
                setDraftName('');
              }
            }}
          />
        )}
      </div>
      <div className={s.foot}>v0.1 · React 18 + TS</div>
    </aside>
  );
}

interface ProjectRowProps {
  slug: string;
  title: string;
  currentFolderId: string | null;
  folders: ReadonlyArray<{ id: string; name: string }>;
  menuOpen: boolean;
  onToggleMenu: () => void;
  onCloseMenu: () => void;
  onMove: (folderId: string | null) => void;
}

function ProjectRow({
  slug,
  title,
  currentFolderId,
  folders,
  menuOpen,
  onToggleMenu,
  onCloseMenu,
  onMove,
}: ProjectRowProps) {
  const btnRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const [menuPos, setMenuPos] = useState<CSSProperties | null>(null);

  // The menu is position: fixed so it escapes the sidebar's overflow clipping,
  // but that means we place it ourselves. On open, measure the trigger and the
  // menu, flip up if it would overflow the viewport bottom, and clamp into the
  // viewport. useLayoutEffect runs before paint, so there's no flash at the
  // wrong position.
  useLayoutEffect(() => {
    if (!menuOpen) {
      setMenuPos(null);
      return;
    }
    const btn = btnRef.current;
    const menu = menuRef.current;
    if (!btn || !menu) return;
    const r = btn.getBoundingClientRect();
    const m = menu.getBoundingClientRect();
    const PAD = 6;
    let top = r.bottom + 4;
    if (top + m.height > window.innerHeight - PAD) top = r.top - m.height - 4;
    if (top < PAD) top = PAD;
    let left = r.left;
    if (left + m.width > window.innerWidth - PAD) left = window.innerWidth - m.width - PAD;
    if (left < PAD) left = PAD;
    setMenuPos({ top, left });
  }, [menuOpen]);

  return (
    <div className={s.projectRow}>
      <Link to={`/project/${slug}`} className={s.link}>
        <Icon name="folder" size={15} className={s.linkIcon} />
        <span className={s.title}>{title}</span>
      </Link>
      <div className={s.menuWrap}>
        <button
          ref={btnRef}
          type="button"
          className={`${s.menuBtn} ${menuOpen ? s.menuBtnOpen : ''}`}
          title="移动到文件夹"
          aria-label="移动到文件夹"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onToggleMenu();
          }}
        >
          ⋯
        </button>
        {menuOpen && (
          <>
            <div className={s.menuBackdrop} onClick={onCloseMenu} />
            <div ref={menuRef} className={s.menu} role="menu" style={menuPos ?? undefined}>
              <button
                type="button"
                role="menuitem"
                className={currentFolderId === null ? s.menuActive : ''}
                onClick={() => {
                  onMove(null);
                  onCloseMenu();
                }}
              >
                未分类
              </button>
              {folders.map((f) => (
                <button
                  key={f.id}
                  type="button"
                  role="menuitem"
                  className={currentFolderId === f.id ? s.menuActive : ''}
                  onClick={() => {
                    onMove(f.id);
                    onCloseMenu();
                  }}
                >
                  {f.name}
                </button>
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
