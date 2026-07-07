export type PetalId = 'known' | 'life' | 'boundary' | 'action' | 'creation';

export interface FlowerPetal {
  id: PetalId;
  label: string;
  color: string;
  softColor: string;
  angle: number;
  flyX: number;
  flyY: number;
  short: string;
}

export interface KnowledgeFlower {
  id: string;
  projectSlug: string;
  title: string;
  planted: boolean;
  plantedAt?: string;
  updatedAt?: string;
  petalContent: Record<PetalId, string>;
}

export const flowerPetals: FlowerPetal[] = [
  {
    id: 'known',
    label: '旧知',
    color: '#9ADBB7',
    softColor: '#DDF2E7',
    angle: 0,
    flyX: -170,
    flyY: -80,
    short: '把新知识接回旧地图',
  },
  {
    id: 'life',
    label: '生活',
    color: '#FFB3A7',
    softColor: '#FFE2DC',
    angle: 72,
    flyX: 190,
    flyY: -70,
    short: '找到今天能看见的例子',
  },
  {
    id: 'boundary',
    label: '边界',
    color: '#A7D9FF',
    softColor: '#E2F2FF',
    angle: 144,
    flyX: 150,
    flyY: 150,
    short: '知道什么时候别用',
  },
  {
    id: 'action',
    label: '行动',
    color: '#FFD875',
    softColor: '#FFF0C6',
    angle: 216,
    flyX: -130,
    flyY: 170,
    short: '变成一个小动作',
  },
  {
    id: 'creation',
    label: '创作',
    color: '#C9B8FF',
    softColor: '#ECE6FF',
    angle: 288,
    flyX: -210,
    flyY: 30,
    short: '产出可分享表达',
  },
];

export const emptyPetalContent: Record<PetalId, string> = {
  known: '',
  life: '',
  boundary: '',
  action: '',
  creation: '',
};

export function createEmptyFlower(projectSlug: string, projectTitle: string): KnowledgeFlower {
  return {
    id: projectSlug,
    projectSlug,
    title: projectTitle || projectSlug,
    planted: false,
    petalContent: { ...emptyPetalContent },
  };
}

export function completedPetalsFor(flower: KnowledgeFlower): PetalId[] {
  return flowerPetals
    .filter((petal) => flower.petalContent[petal.id]?.trim())
    .map((petal) => petal.id);
}

export function completenessFor(flower: KnowledgeFlower): number {
  return completedPetalsFor(flower).length / flowerPetals.length;
}

export function normalizeFlower(
  raw: unknown,
  projectSlug: string,
  projectTitle: string,
): KnowledgeFlower {
  if (!raw || typeof raw !== 'object') return createEmptyFlower(projectSlug, projectTitle);
  const input = raw as Partial<KnowledgeFlower>;
  const rawContent: Partial<Record<PetalId, string>> =
    input.petalContent && typeof input.petalContent === 'object'
      ? input.petalContent
      : {};

  return {
    id: typeof input.id === 'string' && input.id.trim() ? input.id : projectSlug,
    projectSlug,
    title: typeof input.title === 'string' && input.title.trim() ? input.title : projectTitle || projectSlug,
    planted: Boolean(input.planted),
    plantedAt: typeof input.plantedAt === 'string' ? input.plantedAt : undefined,
    updatedAt: typeof input.updatedAt === 'string' ? input.updatedAt : undefined,
    petalContent: flowerPetals.reduce((acc, petal) => {
      acc[petal.id] = typeof rawContent[petal.id] === 'string'
        ? rawContent[petal.id] ?? ''
        : '';
      return acc;
    }, { ...emptyPetalContent }),
  };
}

export function parseFlowerJson(
  source: string,
  projectSlug: string,
  projectTitle: string,
): KnowledgeFlower {
  try {
    return normalizeFlower(JSON.parse(source), projectSlug, projectTitle);
  } catch {
    return createEmptyFlower(projectSlug, projectTitle);
  }
}

export function serializeFlower(flower: KnowledgeFlower): string {
  return `${JSON.stringify(flower, null, 2)}\n`;
}

export function formatFlowerDate(value?: string): string {
  if (!value) return '未种植';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
  });
}
