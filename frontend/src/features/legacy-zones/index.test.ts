import { expect, it } from 'vitest';

import * as legacyZones from './index';

it('does not expose retired Summary, Extend, or Knowledge Garden APIs', () => {
  expect(legacyZones).not.toHaveProperty('SummaryPage');
  expect(legacyZones).not.toHaveProperty('ExtendPage');
  expect(legacyZones).not.toHaveProperty('KnowledgeFlowerGlyph');
  expect(legacyZones).not.toHaveProperty('useFlashcards');
  expect(legacyZones).not.toHaveProperty('useExtendFlower');
  expect(legacyZones).not.toHaveProperty('useSaveExtendFlower');
});
