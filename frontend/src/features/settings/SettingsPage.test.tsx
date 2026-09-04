import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

import { SettingsPage } from './SettingsPage';
import type { AskConfig } from '@/features/settings/askAi';

// Track the save mutation so the test can assert it was called.
const saveMutate = vi.fn();
const probeMutate = vi.fn();

// Hoisted to a stable reference — the real useQuery returns a referentially
// stable `data` between renders, and the component's draft-seeding effect keys
// on `data`. A fresh literal per call would loop the effect infinitely.
const sampleConfig: AskConfig = {
  default: 'p1',
  searchEngine: 'google',
  providers: [
    { id: 'p1', kind: 'openai', name: 'GPT-4o', baseURL: 'https://api.openai.com/v1', apiKey: '••••', model: 'gpt-4o' },
    { id: 'p2', kind: 'anthropic', name: 'Claude', baseURL: 'https://api.anthropic.com', apiKey: '••••', model: 'claude-sonnet-4', thinking: true },
  ],
};

vi.mock('@/features/settings/settings', () => ({
  useAgentRuntimeSettings: () => ({ data: { providers: [], selected: undefined }, isLoading: false }),
  useUpdateAgentRuntime: () => ({ mutate: vi.fn(), isPending: false, isError: false }),
}));

vi.mock('@/features/settings/askAi', () => ({
  useAskAiSettings: () => ({ data: sampleConfig, isLoading: false, isError: false }),
  useSaveAskAiSettings: () => ({ mutate: saveMutate, isPending: false, isError: false }),
  useProbeAskAi: () => ({ mutate: probeMutate, isPending: false, isError: false, data: undefined, variables: undefined }),
}));

function withProviders(ui: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } } });
  return <QueryClientProvider client={client}>{ui}</QueryClientProvider>;
}

describe('SettingsPage — search engine section', () => {
  beforeEach(() => {
    saveMutate.mockReset();
    probeMutate.mockReset();
  });

  it('keeps search engine configuration in unified settings', async () => {
    render(withProviders(<SettingsPage />));
    expect(await screen.findByRole('heading', { name: '搜索引擎' })).toBeTruthy();
    expect(screen.getByLabelText('默认搜索引擎')).toHaveValue('google');
    expect(screen.queryByDisplayValue('GPT-4o')).toBeNull();
  });

  it('changes only the search engine and preserves model connections', async () => {
    render(withProviders(<SettingsPage />));
    fireEvent.change(await screen.findByLabelText('默认搜索引擎'), { target: { value: 'bing' } });

    await waitFor(() => expect(saveMutate).toHaveBeenCalledTimes(1));
    const saved = saveMutate.mock.calls[0][0] as AskConfig;
    expect(saved.searchEngine).toBe('bing');
    expect(saved.default).toBe('p1');
    expect(saved.providers[0].model).toBe('gpt-4o');
    expect(saved.providers[1].name).toBe('Claude');
  });
});
