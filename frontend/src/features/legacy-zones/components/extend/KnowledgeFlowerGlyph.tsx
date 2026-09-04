import { type CSSProperties, useEffect, useRef } from 'react';
import { animate, createScope } from 'animejs';
import clsx from 'clsx';

import { flowerPetals, type PetalId } from './flowerData';
import s from './KnowledgeFlowerGlyph.module.css';

interface KnowledgeFlowerGlyphProps {
  title: string;
  centerLabel?: string;
  completedPetals: ReadonlyArray<PetalId>;
  selectedPetalId?: PetalId;
  onPetalSelect?: (petalId: PetalId) => void;
  size?: 'large' | 'medium' | 'small';
  showLabels?: boolean;
  planted?: boolean;
  showBotany?: boolean;
}

export function KnowledgeFlowerGlyph({
  title,
  centerLabel,
  completedPetals,
  selectedPetalId,
  onPetalSelect,
  size = 'medium',
  showLabels = true,
  planted = false,
  showBotany = true,
}: KnowledgeFlowerGlyphProps) {
  const rootRef = useRef<HTMLDivElement>(null);
  const previousCompleted = useRef(new Set<PetalId>(completedPetals));
  const completed = new Set(completedPetals);

  useEffect(() => {
    if (!rootRef.current) return;
    if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) return;
    const scope = createScope({ root: rootRef.current }).add((self) => {
      if (!self) return;
      animate(`.${s.petalLayer}`, {
        rotate: ['-0.7deg', '0.7deg'],
        duration: 2600,
        loop: true,
        alternate: true,
        ease: 'inOutSine',
      });
      animate(`.${s.stemLayer}`, {
        rotate: ['0.35deg', '-0.35deg'],
        duration: 2800,
        loop: true,
        alternate: true,
        ease: 'inOutSine',
      });
    });
    return () => scope.revert();
  }, []);

  useEffect(() => {
    if (!rootRef.current) return;
    const previous = previousCompleted.current;
    const next = new Set(completedPetals);
    const entered = completedPetals.filter((id) => !previous.has(id));
    previousCompleted.current = next;
    if (entered.length === 0) return;
    if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) return;

    const scope = createScope({ root: rootRef.current }).add((self) => {
      if (!self) return;
      entered.forEach((id, index) => {
        const petal = flowerPetals.find((item) => item.id === id);
        if (!petal) return;
        animate(`[data-petal="${id}"] .${s.petalFace}`, {
          opacity: [0, 1],
          '--petal-fly-x': [`${petal.flyX}px`, '0px'],
          '--petal-fly-y': [`${petal.flyY}px`, '0px'],
          '--petal-motion-scale': [0.42, 1.08, 1],
          '--petal-motion-rotate': ['-7deg', '3deg', '0deg'],
          duration: 680,
          delay: index * 95,
          ease: 'outBack',
        });
        animate(`[data-petal="${id}"] .${s.petalGlow}`, {
          opacity: [0, 0.65, 0],
          '--petal-glow-scale': [0.9, 1.35],
          duration: 760,
          delay: index * 95 + 110,
          ease: 'outQuad',
        });
      });
      animate(`.${s.core}`, {
        '--core-scale': [0.96, 1.04, 1],
        duration: 520,
        delay: entered.length * 70,
        ease: 'outQuad',
      });
    });

    return () => scope.revert();
  }, [completedPetals]);

  return (
    <div
      ref={rootRef}
      className={clsx(s.flower, s[size], planted && s.planted)}
      aria-label={`${title} 知识小花`}
    >
      {showBotany && (
        <div className={s.stemLayer} aria-hidden="true">
          <span className={s.stem} />
          <span className={clsx(s.leaf, s.leafLeft)} />
          <span className={clsx(s.leaf, s.leafRight)} />
          <span className={s.sepal} />
        </div>
      )}
      <div className={s.petalLayer}>
        {flowerPetals.map((petal) => {
          const isComplete = completed.has(petal.id);
          const isSelected = selectedPetalId === petal.id;
          return (
            <button
              key={petal.id}
              type="button"
              data-petal={petal.id}
              className={clsx(
                s.petalSlot,
                isComplete && s.petalComplete,
                isSelected && s.petalSelected,
              )}
              style={{
                '--petal-angle': `${petal.angle}deg`,
                '--petal-color': petal.color,
                '--petal-soft': petal.softColor,
              } as CSSProperties}
              onClick={(event) => {
                event.stopPropagation();
                onPetalSelect?.(petal.id);
              }}
              aria-pressed={isSelected}
              aria-label={`${petal.label}${isComplete ? '已补齐' : '待补齐'}`}
            >
              <span className={s.petalGlow} />
              <span className={s.petalFace}>
                {showLabels && <span className={s.petalLabel}>{petal.label}</span>}
              </span>
            </button>
          );
        })}
      </div>
      <button
        type="button"
        className={s.core}
        onClick={(event) => {
          event.stopPropagation();
          onPetalSelect?.(selectedPetalId ?? completedPetals[0] ?? 'known');
        }}
        aria-label={`${title} 知识核`}
      >
        <span>{centerLabel ?? title}</span>
      </button>
    </div>
  );
}
