import { useQuery } from '@tanstack/react-query';
import { http } from '@/shared/client';
import type { DisciplineTopicsResponse } from '@/shared/types/api';
export interface DisciplineMapProjection { mapVersionId:string; format:'structured'|'legacy'; overviewMarkdown:string; topics:DisciplineTopicsResponse; coverage?:unknown }
export function useDisciplineMap(slug:string) {
 return useQuery({queryKey:['projects',slug,'discipline-map'],queryFn:()=>http.get<DisciplineMapProjection>(`/api/projects/${encodeURIComponent(slug)}/discipline-map`)});
}
