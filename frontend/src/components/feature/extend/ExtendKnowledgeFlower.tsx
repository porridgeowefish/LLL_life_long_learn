import { type CSSProperties, useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { animate, createScope, stagger } from 'animejs';
import clsx from 'clsx';

import { useExtendFlower, useSaveExtendFlower } from '@/api/extendFlowers';

import { KnowledgeFlowerGlyph } from './KnowledgeFlowerGlyph';
import {
  completedPetalsFor,
  completenessFor,
  flowerPetalMeanings,
  flowerPetals,
  type KnowledgeFlower,
  type PetalId,
} from './flowerData';
import s from './ExtendKnowledgeFlower.module.css';

const prefersReducedMotion = () => (
  window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
);

function editableSnapshot(flower: KnowledgeFlower | null | undefined) {
  if (!flower) return '';
  return JSON.stringify({
    planted: flower.planted,
    plantedAt: flower.plantedAt ?? '',
    petalContent: flower.petalContent,
  });
}

interface ExtendKnowledgeFlowerProps {
  projectSlug: string;
  projectTitle: string;
}

export function ExtendKnowledgeFlower({
  projectSlug,
  projectTitle,
}: ExtendKnowledgeFlowerProps) {
  const rootRef = useRef<HTMLElement>(null);
  const { data: savedFlower, isLoading, error } = useExtendFlower(projectSlug, projectTitle);
  const saveFlower = useSaveExtendFlower();
  const [draft, setDraft] = useState<KnowledgeFlower | null>(null);
  const [selectedPetalId, setSelectedPetalId] = useState<PetalId>('known');

  useEffect(() => {
    if (savedFlower) setDraft(savedFlower);
  }, [savedFlower]);

  useEffect(() => {
    if (!rootRef.current) return;
    if (prefersReducedMotion()) return;
    const scope = createScope({ root: rootRef.current }).add((self) => {
      if (!self) return;
      animate('[data-extend-seed]', {
        opacity: [0, 1],
        y: [12, 0],
        delay: stagger(50),
        duration: 380,
        ease: 'outQuad',
      });
    });
    return () => scope.revert();
  }, []);

  const flower = draft ?? savedFlower;
  const selectedPetal = flowerPetals.find((petal) => petal.id === selectedPetalId) ?? flowerPetals[0];
  const completedPetals = useMemo(
    () => (flower ? completedPetalsFor(flower) : []),
    [flower],
  );
  const completeness = flower ? completenessFor(flower) : 0;
  const isComplete = completedPetals.length === flowerPetals.length;
  const dirty = editableSnapshot(flower) !== editableSnapshot(savedFlower);

  const updatePetal = (petalId: PetalId, value: string) => {
    setDraft((current) => {
      if (!current) return current;
      return {
        ...current,
        planted: false,
        plantedAt: undefined,
        petalContent: {
          ...current.petalContent,
          [petalId]: value,
        },
      };
    });
  };

  const save = async (nextFlower = flower) => {
    if (!nextFlower) return;
    await saveFlower.mutateAsync({ flower: nextFlower });
  };

  const plant = async () => {
    if (!flower || !isComplete) return;
    const nextFlower: KnowledgeFlower = {
      ...flower,
      planted: true,
      plantedAt: flower.plantedAt ?? new Date().toISOString(),
    };
    setDraft(nextFlower);
    await save(nextFlower);
  };

  if (isLoading || !flower) {
    return <div className={s.loading}>正在打开这朵拓展小花…</div>;
  }

  if (error) {
    return (
      <div className={s.errorBox}>
        小花加载失败：{(error as Error).message}
      </div>
    );
  }

  return (
    <section ref={rootRef} className={s.root} aria-label={`${projectTitle} 的拓展小花`}>
      <div className={s.head}>
        <div>
          <p className={s.kicker}>第四模块 · 拓展</p>
          <h2 className={s.title}>把「{projectTitle}」补成一朵小花</h2>
        </div>
        <div className={s.actions}>
          <button
            type="button"
            className={s.secondaryButton}
            onClick={() => setDraft(savedFlower ?? flower)}
            disabled={!dirty || saveFlower.isPending}
          >
            放弃修改
          </button>
          <button
            type="button"
            className={s.primaryButton}
            onClick={() => void save()}
            disabled={!dirty || saveFlower.isPending}
          >
            {saveFlower.isPending ? '保存中…' : dirty ? '保存小花' : '已保存'}
          </button>
        </div>
      </div>

      <div className={s.stageGrid}>
        <div className={s.stage} data-extend-seed>
          <div className={s.sourceRail} aria-label="可编辑花瓣节点">
            {flowerPetals.map((petal) => {
              const done = completedPetals.includes(petal.id);
              return (
                <button
                  key={petal.id}
                  type="button"
                  data-source={petal.id}
                  className={clsx(
                    s.sourceChip,
                    done && s.sourceChipDone,
                    selectedPetalId === petal.id && s.sourceChipActive,
                  )}
                  style={{ '--petal-color': petal.color, '--petal-soft': petal.softColor } as CSSProperties}
                  onClick={() => setSelectedPetalId(petal.id)}
                >
                  <span className={s.sourceDot} />
                  <span>{petal.label}</span>
                  <small>{done ? '已补齐' : '待填写'}</small>
                </button>
              );
            })}
          </div>

          <div className={s.flowerDock}>
            <KnowledgeFlowerGlyph
              title={flower.title}
              centerLabel={flower.planted ? '已种' : isComplete ? '可种植' : '知识核'}
              completedPetals={completedPetals}
              selectedPetalId={selectedPetalId}
              onPetalSelect={setSelectedPetalId}
              size="large"
            />
          </div>

          <div className={s.dimensionGuide} aria-label="知识花五个维度">
            {flowerPetals.map((petal) => (
              <button
                key={petal.id}
                type="button"
                className={clsx(s.dimensionItem, selectedPetalId === petal.id && s.dimensionItemActive)}
                style={{ '--petal-color': petal.color } as CSSProperties}
                onClick={() => setSelectedPetalId(petal.id)}
              >
                <strong>{petal.label}</strong>
                <span>{flowerPetalMeanings[petal.id]}</span>
              </button>
            ))}
          </div>

          <div className={s.gardenHint}>
            <span>{Math.round(completeness * 100)}%</span>
            <p>
              {isComplete
                ? flower.planted
                  ? '这朵小花已经种进知识温室。'
                  : '五片花瓣都补齐了，可以种进知识温室。'
                : '写完一个节点，淡色花瓣会飞入并变成彩色。'}
            </p>
          </div>
        </div>

        <aside className={s.panel} data-extend-seed>
          <p className={s.modeLabel}>当前节点 · {selectedPetal.label}</p>
          <h3 className={s.panelTitle}>{selectedPetal.short}</h3>

          <label className={s.editorLabel} htmlFor={`flower-${selectedPetal.id}`}>
            花瓣内容
          </label>
          <textarea
            id={`flower-${selectedPetal.id}`}
            className={s.textarea}
            value={flower.petalContent[selectedPetal.id]}
            onChange={(event) => updatePetal(selectedPetal.id, event.target.value)}
            placeholder={`写下「${selectedPetal.label}」这一瓣要反哺给你的内容…`}
          />

          <div className={s.selectedPreview}>
            <div
              className={s.selectedMark}
              style={{ '--petal-color': selectedPetal.color } as CSSProperties}
            >
              {selectedPetal.label}
            </div>
            <p>{flower.petalContent[selectedPetal.id] || '这个节点还空着。'}</p>
          </div>

          <div className={s.plantActions}>
            <button
              type="button"
              className={s.plantButton}
              onClick={() => void plant()}
              disabled={!isComplete || flower.planted || saveFlower.isPending}
            >
              {flower.planted ? '已种到温室' : isComplete ? '种到知识温室' : '补齐五瓣后可种植'}
            </button>
            <Link to="/greenhouse" className={s.greenhouseLink}>
              查看知识温室
            </Link>
          </div>

          <div className={s.microCopy}>
            温室只浏览和回看；每片花瓣的编辑都在当前学习单元完成。
          </div>

          {saveFlower.isError && (
            <div className={s.errorText}>保存失败：{(saveFlower.error as Error).message}</div>
          )}
        </aside>
      </div>
    </section>
  );
}
