import { fireEvent, render, screen, waitFor } from '@testing-library/react';

import { AgentInvokePanel } from './AgentInvokePanel';

const mutateAsync = vi.fn();
const resumeMutateAsync = vi.fn();
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
        {
          id: 'explain',
          name: 'Explain Agent',
          icon: 'E',
          description: 'explain',
          userStory: 'help me understand',
          allowedZones: ['Explain'],
          primitives: { required: [], optional: [] },
          charterPath: 'agents/charters/explain.md',
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
  useResumeExplainSession: () => ({
    mutateAsync: resumeMutateAsync,
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
    resumeMutateAsync.mockReset();
    setActiveSession.mockReset();
    mutateAsync.mockResolvedValue({ session: { id: 'sess-1' }, runDir: 'runs/test' });
    resumeMutateAsync.mockResolvedValue({
      resumed: true,
      session: { id: 'sess-explain' },
      runDir: 'runs/resume',
    });
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
    expect(screen.queryByRole('button', { name: '继续上次会话' })).not.toBeInTheDocument();
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

  it('resumes the previous Explain session with a distinct button', async () => {
    render(<AgentInvokePanel slug="demo" zone="Explain" />);

    fireEvent.click(screen.getByRole('button', { name: '继续上次会话' }));

    await waitFor(() => expect(resumeMutateAsync).toHaveBeenCalledTimes(1));
    expect(resumeMutateAsync).toHaveBeenCalledWith('demo');
    expect(mutateAsync).not.toHaveBeenCalled();
    expect(setActiveSession).toHaveBeenCalledWith('sess-explain');
  });
});
