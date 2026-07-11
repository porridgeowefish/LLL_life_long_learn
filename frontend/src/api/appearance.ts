import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { http } from './client';
import { qk } from './queryKeys';
import { applyTheme, type ThemePreference } from '@/lib/theme';

export interface AppearanceConfig { theme: ThemePreference }

export function useAppearance() {
  return useQuery({
    queryKey: qk.settings.appearance(),
    queryFn: async () => {
      const config = await http.get<AppearanceConfig>('/api/settings/appearance');
      applyTheme(config.theme);
      return config;
    },
  });
}

export function useUpdateAppearance() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (theme: ThemePreference) => {
      applyTheme(theme);
      return http.put<AppearanceConfig>('/api/settings/appearance', { theme });
    },
    onSuccess: (data) => qc.setQueryData(qk.settings.appearance(), data),
  });
}
