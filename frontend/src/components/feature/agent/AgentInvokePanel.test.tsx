import { fireEvent, render, screen, waitFor } from '@testing-library/react';

import { AgentInvokePanel } from './AgentInvokePanel';

const mutateAsync = vi.fn();
const setActiveSession = vi.fn();

vi.mock('@/api/agents', () => ({
  useAgents: () => ({
    data: {
      agents: [
        {
          id: 'intro',
          name: 'Intro Agent',
          icon: 'I',
          description: 'intro',
          userStory: 'help me start',
          allowedZones: ['Intro'],
          primitives: { required: [], optional: [] },
          charterPath: 'agents/charters/intro.md',
          defaultOutputTargets: [],
        },
      ],
    },
  }),
  useInvokeAgent: () => ({
    mutateAsync,
    isPending: false,
    error: null,
  }),
}));

vi.mock('@/store/slices/session', () => ({
  useSessionStore: (selector: (state: { setActiveSession: typeof setActiveSession }) => unknown) =>
    selector({ setActiveSession }),
}));

describe('AgentInvokePanel', () => {
  beforeEach(() => {
    mutateAsync.mockReset();
    setActiveSession.mockReset();
    mutateAsync.mockResolvedValue({ session: { id: 'sess-1' }, runDir: 'runs/test' });
  });

  it('invokes without additional guidance by default', async () => {
    render(<AgentInvokePanel slug="demo" zone="Intro" />);

    fireEvent.click(screen.getByRole('button', { name: '调用' }));

    await waitFor(() => expect(mutateAsync).toHaveBeenCalledTimes(1));
    expect(mutateAsync).toHaveBeenCalledWith({
      agentId: 'intro',
      payload: expect.objectContaining({
        projectId: 'demo',
        zone: 'Intro',
        intent: undefined,
        permissionMode: 'auto',
      }),
    });
    expect(setActiveSession).toHaveBeenCalledWith('sess-1');
  });

  it('sends optional guidance when the field is expanded', async () => {
    render(<AgentInvokePanel slug="demo" zone="Intro" />);

    fireEvent.click(screen.getByRole('button', { name: '补充说明（可选）' }));
    fireEvent.change(screen.getByLabelText('补充说明'), {
      target: { value: '先帮我搭一个最短入门路径' },
    });
    fireEvent.click(screen.getByRole('button', { name: '调用' }));

    await waitFor(() => expect(mutateAsync).toHaveBeenCalledTimes(1));
    expect(mutateAsync).toHaveBeenCalledWith({
      agentId: 'intro',
      payload: expect.objectContaining({
        intent: '先帮我搭一个最短入门路径',
      }),
    });
  });
});
