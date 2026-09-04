import { describe, it, expect } from 'vitest';

import {
  assignProject,
  createFolder,
  deleteFolder,
  folderOf,
  renameFolder,
  uncategorizedSlugs,
  type FolderLayout,
} from './folders';

const layout: FolderLayout = {
  folders: [
    { id: 'a', name: '编程', slugOrder: ['go'] },
    { id: 'b', name: '社科', slugOrder: ['econ', 'hist'] },
  ],
};

describe('folder helpers', () => {
  it('uncategorized excludes assigned slugs', () => {
    expect(uncategorizedSlugs(layout, ['go', 'econ', 'hist', 'rust'])).toEqual(['rust']);
  });

  it('folderOf finds the owning folder or null', () => {
    expect(folderOf(layout, 'econ')).toBe('b');
    expect(folderOf(layout, 'rust')).toBeNull();
  });

  it('assignProject moves between folders (tree membership)', () => {
    const next = assignProject(layout, 'go', 'b');
    expect(folderOf(next, 'go')).toBe('b');
    expect(next.folders[0].slugOrder).toEqual([]);
  });

  it('assignProject(null) removes from all folders', () => {
    const next = assignProject(layout, 'econ', null);
    expect(folderOf(next, 'econ')).toBeNull();
  });

  it('does not mutate the input layout', () => {
    const before = JSON.parse(JSON.stringify(layout));
    assignProject(layout, 'go', 'b');
    renameFolder(layout, 'a', 'x');
    deleteFolder(layout, 'a');
    expect(layout).toEqual(before);
  });

  it('createFolder ignores empty names', () => {
    expect(createFolder(layout, '  ').folders.length).toBe(2);
    expect(createFolder(layout, '新').folders.length).toBe(3);
  });

  it('renameFolder ignores empty names', () => {
    expect(renameFolder(layout, 'a', '  ').folders[0].name).toBe('编程');
  });

  it('deleteFolder drops the folder', () => {
    expect(deleteFolder(layout, 'a').folders.length).toBe(1);
  });
});
