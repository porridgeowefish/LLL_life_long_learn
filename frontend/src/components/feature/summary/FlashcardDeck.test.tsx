import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { FlashcardDeck } from './FlashcardDeck';

const gradeMutate = vi.fn();

vi.mock('@/api/flashcards', () => ({
  useFlashcards: () => ({
    isLoading: false,
    isError: false,
    data: {
      flashcards: [
        {
          id: 'fc-001',
          front: '为什么 NFA 与 DFA 表达能力相同？',
          back: '子集构造保持识别语言不变。',
          category: 'relationship',
          sourceRefs: ['explain/pages/005-nfa-dfa.md'],
        },
        {
          id: 'fc-002',
          front: '何时使用优先级？',
          back: '最长匹配长度相同时。',
          category: 'boundary',
          sourceRefs: ['explain/pages/007-scanning-rules.md'],
        },
      ],
      progress: [],
    },
  }),
  useGradeFlashcard: () => ({
    mutate: gradeMutate,
    isPending: false,
  }),
}));

vi.mock('@/hooks/useMarkdown', () => ({
  useMarkdown: (input: string) => ({ html: `<p>${input}</p>`, mermaid: [] }),
}));

describe('FlashcardDeck', () => {
  beforeEach(() => gradeMutate.mockReset());

  it('reveals a reasoning answer and keeps navigation compact', () => {
    render(<FlashcardDeck projectSlug="lexer" />);

    expect(screen.getByText('为什么 NFA 与 DFA 表达能力相同？')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /显示答案/ }));
    expect(screen.getByText('子集构造保持识别语言不变。')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /有点模糊/ })).toBeInTheDocument();
  });

  it('supports keyboard navigation between cards', () => {
    render(<FlashcardDeck projectSlug="lexer" />);
    fireEvent.keyDown(window, { key: 'ArrowRight' });
    expect(screen.getByText('何时使用优先级？')).toBeInTheDocument();
  });
});
