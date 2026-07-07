import { type CSSProperties, useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { animate, createScope, stagger } from 'animejs';
import clsx from 'clsx';

import { useProjectFlowers } from '@/api/extendFlowers';
import { useProjects } from '@/api/projects';
import { KnowledgeFlowerGlyph } from '@/components/feature/extend/KnowledgeFlowerGlyph';
import {
  completedPetalsFor,
  completenessFor,
  flowerPetals,
  formatFlowerDate,
  type KnowledgeFlower,
  type PetalId,
} from '@/components/feature/extend/flowerData';

import s from './GreenhousePage.module.css';

const prefersReducedMotion = () => (
  window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
);

export function GreenhousePage() {
  const rootRef = useRef<HTMLDivElement>(null);
  const { data: projectsData, isLoading: projectsLoading } = useProjects();
  const projects = projectsData?.projects ?? [];
  const flowerQueries = useProjectFlowers(projects);
  const greenhouseFlowers = flowerQueries
    .map((query) => query.data)
    .filter((flower): flower is KnowledgeFlower => Boolean(flower?.planted));

  const [selectedFlowerId, setSelectedFlowerId] = useState('');
  const [selectedPetalId, setSelectedPetalId] = useState<PetalId>('known');
  const selectedFlower = greenhouseFlowers.find((flower) => flower.id === selectedFlowerId) ?? greenhouseFlowers[0];
  const selectedPetal = flowerPetals.find((petal) => petal.id === selectedPetalId) ?? flowerPetals[0];
  const selectedCompletedPetals = selectedFlower ? completedPetalsFor(selectedFlower) : [];
  const isLoading = projectsLoading || flowerQueries.some((query) => query.isLoading);

  useEffect(() => {
    if (selectedFlower && selectedFlower.id === selectedFlowerId) return;
    if (!greenhouseFlowers[0]) return;
    setSelectedFlowerId(greenhouseFlowers[0].id);
    setSelectedPetalId(completedPetalsFor(greenhouseFlowers[0])[0] ?? 'known');
  }, [greenhouseFlowers, selectedFlower, selectedFlowerId]);

  useEffect(() => {
    if (!rootRef.current) return;
    if (prefersReducedMotion()) return;
    const scope = createScope({ root: rootRef.current }).add((self) => {
      if (!self) return;
      animate('[data-bed-flower]', {
        opacity: [0, 1],
        '--plot-enter': ['28px', '0px'],
        delay: stagger(95, { from: 'center' }),
        duration: 620,
        ease: 'outBack',
      });
      animate('[data-greenhouse-panel]', {
        opacity: [0, 1],
        x: [18, 0],
        duration: 460,
        delay: 180,
        ease: 'outQuad',
      });
    });
    return () => scope.revert();
  }, []);

  useEffect(() => {
    if (!rootRef.current) return;
    if (prefersReducedMotion()) return;
    animate('[data-greenhouse-detail]', {
      opacity: [0, 1],
      y: [8, 0],
      duration: 280,
      ease: 'outQuad',
    });
  }, [selectedFlowerId, selectedPetalId]);

  const selectFlower = (flower: KnowledgeFlower) => {
    setSelectedFlowerId(flower.id);
    setSelectedPetalId(completedPetalsFor(flower)[0] ?? 'known');
  };
  const plotDepths = [
    { shift: 18, scale: 1.05 },
    { shift: -14, scale: 0.95 },
    { shift: 6, scale: 1 },
    { shift: -20, scale: 0.92 },
  ];

  return (
    <div className={s.root} ref={rootRef}>
      <header className={s.header}>
        <div>
          <p className={s.kicker}>知识温室</p>
          <h1 className={s.title}>已经补齐的小花，种在这里慢慢长大</h1>
        </div>
        {selectedFlower && (
          <Link to={`/project/${selectedFlower.projectSlug}/Extend`} className={s.backLink}>
            回到这朵小花编辑
          </Link>
        )}
      </header>

      <div className={s.layout}>
        <section className={s.greenhouse} aria-label="知识小花苗床">
          <div className={s.glassRoof} />
          <div className={s.bed}>
            {isLoading && <div className={s.emptyBed}>正在整理温室里的小花…</div>}
            {!isLoading && greenhouseFlowers.length === 0 && (
              <div className={s.emptyBed}>
                还没有种下的小花。回到某个学习单元的拓展阶段，补齐五片花瓣后再种进来。
              </div>
            )}
            {greenhouseFlowers.map((flower, index) => {
              const completedPetals = completedPetalsFor(flower);
              const completeness = completenessFor(flower);
              return (
              <div
                key={flower.id}
                data-bed-flower
                className={clsx(
                  s.flowerPlot,
                  selectedFlower.id === flower.id && s.flowerPlotActive,
                )}
                style={{
                  '--plot-shift': `${plotDepths[index % plotDepths.length].shift}px`,
                  '--plot-scale': plotDepths[index % plotDepths.length].scale,
                } as CSSProperties}
                onClick={() => selectFlower(flower)}
              >
                <KnowledgeFlowerGlyph
                  title={flower.title}
                  centerLabel={completeness === 1 ? '已种' : '发芽'}
                  completedPetals={completedPetals}
                  selectedPetalId={selectedFlower.id === flower.id ? selectedPetalId : undefined}
                  onPetalSelect={(petalId) => {
                    setSelectedFlowerId(flower.id);
                    setSelectedPetalId(petalId);
                  }}
                  size="small"
                  showLabels={false}
                  planted
                />
                <span className={s.plotTitle}>{flower.title}</span>
                <span className={s.plotMeta}>{Math.round(completeness * 100)}% · {formatFlowerDate(flower.plantedAt)}</span>
              </div>
            );
            })}
          </div>
        </section>

        {selectedFlower ? (
          <aside className={s.panel} data-greenhouse-panel>
          <p className={s.panelKicker}>已种植 · {formatFlowerDate(selectedFlower.plantedAt)}</p>
          <h2 className={s.panelTitle}>{selectedFlower.title}</h2>
          <p className={s.panelMood}>这朵小花来自「{selectedFlower.title}」学习单元的拓展阶段。</p>

          <div className={s.petalTabs} aria-label="花瓣内容">
            {flowerPetals.map((petal) => {
              const done = selectedCompletedPetals.includes(petal.id);
              return (
                <button
                  key={petal.id}
                  type="button"
                  className={clsx(
                    s.petalTab,
                    selectedPetal.id === petal.id && s.petalTabActive,
                    !done && s.petalTabMuted,
                  )}
                  style={{ '--petal-color': petal.color } as CSSProperties}
                  onClick={() => setSelectedPetalId(petal.id)}
                >
                  {petal.label}
                </button>
              );
            })}
          </div>

          <div className={s.detail} data-greenhouse-detail>
            <div
              className={s.detailMark}
              style={{ '--petal-color': selectedPetal.color } as CSSProperties}
            >
              {selectedPetal.label}
            </div>
            <div>
              <p className={s.detailTitle}>{selectedPetal.short}</p>
              <p className={s.detailBody}>{selectedFlower.petalContent[selectedPetal.id]}</p>
            </div>
          </div>

          <Link to={`/project/${selectedFlower.projectSlug}/Extend`} className={s.editLink}>
            回到学习单元编辑
          </Link>

          <div className={s.editNote}>
            温室只查看花和花瓣内容；要改内容，请回到对应知识单元的拓展页。
          </div>
        </aside>
        ) : (
          <aside className={clsx(s.panel, s.emptyPanel)} data-greenhouse-panel>
            <p className={s.panelKicker}>等待种植</p>
            <h2 className={s.panelTitle}>温室现在是空的</h2>
            <p className={s.panelMood}>
              每朵小花都属于一个学习单元。补齐五片花瓣并点击种植后，它才会出现在这里。
            </p>
          </aside>
        )}
      </div>
    </div>
  );
}
