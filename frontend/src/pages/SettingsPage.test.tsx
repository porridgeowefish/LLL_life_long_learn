import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

import { SettingsPage } from './SettingsPage';
import type { AskConfig } from '@/api/askAi';

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

vi.mock('@/api/settings', () => ({
  useAgentRuntimeSettings: () => ({ data: { providers: [], selected: undefined }, isLoading: false }),
  useUpdateAgentRuntime: () => ({ mutate: vi.fn(), isPending: false, isError: false }),
}));

vi.mock('@/api/askAi', () => ({
  useAskAiSettings: () => ({ data: sampleConfig, isLoading: false, isError: false }),
  useSaveAskAiSettings: () => ({ mutate: saveMutate, isPending: false, isError: false }),
  useProbeAskAi: () => ({ mutate: probeMutate, isPending: false, isError: false, data: undefined, variables: undefined }),
}));

function withProviders(ui: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } } });
  return <QueryClientProvider client={client}>{ui}</QueryClientProvider>;
}

describe('SettingsPage — Ask-AI section', () => {
  beforeEach(() => {
    saveMutate.mockReset();
    probeMutate.mockReset();
  });

  it('renders the configured providers', async () => {
    render(withProviders(<SettingsPage />));
    // Wait for the draft effect to seed. Provider names render as input values.
    expect(await screen.findByDisplayValue('GPT-4o')).toBeTruthy();
    expect(screen.getByDisplayValue('Claude')).toBeTruthy();
  });

  it('edits a provider field and saves via the mutation', async () => {
    render(withProviders(<SettingsPage />));
    // Wait for seeding.
    await screen.findByDisplayValue('GPT-4o');

    // Edit the first provider's model field (two "Model" inputs exist; pick [0]).
    const modelInputs = screen.getAllByLabelText('Model');
    fireEvent.change(modelInputs[0], { target: { value: 'gpt-4o-mini' } });

    // Click 保存.
    fireEvent.click(screen.getByRole('button', { name: '保存' }));

    await waitFor(() => expect(saveMutate).toHaveBeenCalledTimes(1));
    const saved = saveMutate.mock.calls[0][0] as AskConfig;
    expect(saved.providers[0].model).toBe('gpt-4o-mini');
    // Untouched fields are preserved.
    expect(saved.default).toBe('p1');
    expect(saved.searchEngine).toBe('google');
    expect(saved.providers[1].name).toBe('Claude');
  });
});
