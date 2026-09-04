import { type CSSProperties, useEffect, useRef, useState } from "react";
import {
  ArrowRightIcon,
  CheckCircledIcon,
  QuestionMarkCircledIcon,
  UpdateIcon,
} from "@radix-ui/react-icons";
import { Link } from "react-router-dom";
import { animate, createScope, stagger } from "animejs";
import clsx from "clsx";

import { useProjectFlowers } from "@/features/legacy-zones";
import { useProjects } from "@/features/projects/api/projects";
import { KnowledgeFlowerGlyph } from "@/features/legacy-zones";
import {
  completedPetalsFor,
  completenessFor,
  flowerPetals,
  formatFlowerDate,
  type KnowledgeFlower,
  type PetalId,
} from "@/features/legacy-zones";

import s from "./GreenhousePage.module.css";

const prefersReducedMotion = () =>
  window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;

export function GreenhousePage() {
  const rootRef = useRef<HTMLDivElement>(null);
  const {
    data: projectsData,
    isLoading: projectsLoading,
    isError: projectsError,
  } = useProjects();
  const projects = projectsData?.projects ?? [];
  const flowerQueries = useProjectFlowers(projects);
  const greenhouseFlowers = flowerQueries
    .map((query) => query.data)
    .filter((flower): flower is KnowledgeFlower => Boolean(flower?.planted));

  const [selectedFlowerId, setSelectedFlowerId] = useState("");
  const [selectedPetalId, setSelectedPetalId] = useState<PetalId>("known");
  const selectedFlower =
    greenhouseFlowers.find((flower) => flower.id === selectedFlowerId) ??
    greenhouseFlowers[0];
  const selectedPetal =
    flowerPetals.find((petal) => petal.id === selectedPetalId) ??
    flowerPetals[0];
  const selectedCompletedPetals = selectedFlower
    ? completedPetalsFor(selectedFlower)
    : [];
  const isLoading =
    projectsLoading || flowerQueries.some((query) => query.isLoading);
  const isError = projectsError || flowerQueries.some((query) => query.isError);
  const totalCompletedPetals = greenhouseFlowers.reduce(
    (total, flower) => total + completedPetalsFor(flower).length,
    0,
  );

  useEffect(() => {
    if (selectedFlower && selectedFlower.id === selectedFlowerId) return;
    if (!greenhouseFlowers[0]) return;
    setSelectedFlowerId(greenhouseFlowers[0].id);
    setSelectedPetalId(completedPetalsFor(greenhouseFlowers[0])[0] ?? "known");
  }, [greenhouseFlowers, selectedFlower, selectedFlowerId]);

  useEffect(() => {
    if (!rootRef.current || isLoading) return;
    if (prefersReducedMotion()) return;
    const scope = createScope({ root: rootRef.current }).add((self) => {
      if (!self) return;
      animate("[data-bed-flower]", {
        opacity: [0, 1],
        "--plot-enter": ["22px", "0px"],
        delay: stagger(80, { from: "center" }),
        duration: 560,
        ease: "outBack",
      });
      animate("[data-greenhouse-panel]", {
        opacity: [0, 1],
        x: [14, 0],
        duration: 420,
        delay: 130,
        ease: "outQuad",
      });
    });
    return () => scope.revert();
  }, [greenhouseFlowers.length, isLoading]);

  useEffect(() => {
    if (!rootRef.current) return;
    if (prefersReducedMotion()) return;
    const animation = animate(
      rootRef.current.querySelectorAll("[data-greenhouse-detail]"),
      {
      opacity: [0, 1],
      y: [7, 0],
      duration: 260,
      ease: "outQuad",
      },
    );
    return () => {
      animation.revert();
    };
  }, [selectedFlowerId, selectedPetalId]);

  const selectFlower = (flower: KnowledgeFlower) => {
    setSelectedFlowerId(flower.id);
    setSelectedPetalId(completedPetalsFor(flower)[0] ?? "known");
  };

  const plotDepths = [
    { shift: 12, scale: 1.04 },
    { shift: -8, scale: 0.96 },
    { shift: 4, scale: 1 },
    { shift: -12, scale: 0.94 },
  ];

  return (
    <div className={s.root} ref={rootRef}>
      <header className={s.header}>
        <div className={s.headingCopy}>
          <p className={s.kicker}>知识温室</p>
          <h1 className={s.title}>把学过的知识，培育成随时能用的收藏</h1>
          <p className={s.subtitle}>
            每朵花代表一个学习单元，五片花瓣记录旧知、生活、边界、行动与创作。点击花朵，回看它如何连接到你的真实世界。
          </p>
        </div>
        <div className={s.collectionStatus} aria-label="温室收集进度">
          <span>
            <strong>{greenhouseFlowers.length}</strong> 朵知识花
          </span>
          <span>
            <strong>{totalCompletedPetals}</strong> 片已完成花瓣
          </span>
        </div>
      </header>

      <section className={s.guide} aria-labelledby="greenhouse-guide-title">
        <div className={s.guideIntro}>
          <QuestionMarkCircledIcon aria-hidden="true" />
          <div>
            <h2 id="greenhouse-guide-title">温室怎么玩</h2>
            <p>这里是学习成果陈列室，也是定期回顾知识连接的入口。</p>
          </div>
        </div>
        <ol className={s.guideSteps}>
          <li>
            <span>补齐</span>在学习单元中写完五片花瓣
          </li>
          <li>
            <span>种植</span>完成后把知识花放进温室
          </li>
          <li>
            <span>回看</span>点击花朵检查连接与下一步行动
          </li>
        </ol>
      </section>

      <div className={s.layout}>
        <section
          className={s.greenhouse}
          aria-labelledby="greenhouse-bed-title"
        >
          <div className={s.greenhouseTopbar}>
            <div>
              <p>培育区</p>
              <h2 id="greenhouse-bed-title">我的知识花圃</h2>
            </div>
            <span className={s.climate}>光照适宜 · 适合回顾</span>
          </div>

          <div className={s.glassHouse} aria-hidden="true">
            <span className={s.roofPane} />
            <span className={s.sunPatch} />
            <span className={s.hangingPlant} />
          </div>

          <div className={s.bed}>
            {isLoading && (
              <div className={s.loadingBed} role="status">
                <UpdateIcon aria-hidden="true" />
                <div>
                  <strong>温室管理员正在巡园</strong>
                  <span>正在整理你种下的知识花</span>
                </div>
              </div>
            )}

            {!isLoading && isError && (
              <div className={clsx(s.emptyBed, s.errorBed)} role="alert">
                <strong>暂时没能打开温室</strong>
                <span>
                  请稍后刷新页面。你的知识花仍保存在原来的学习单元中。
                </span>
              </div>
            )}

            {!isLoading && !isError && greenhouseFlowers.length === 0 && (
              <div className={s.emptyBed}>
                <div className={s.emptySprout} aria-hidden="true">
                  <span />
                </div>
                <strong>第一块花圃正等着你</strong>
                <span>
                  进入任意学习单元的拓展阶段，补齐五片花瓣并点击“种植”，第一朵知识花就会在这里发芽。
                </span>
                {projects[0] && (
                  <Link
                    to={`/project/${projects[0].slug}/Extend`}
                    className={s.emptyAction}
                  >
                    去培育第一朵花 <ArrowRightIcon aria-hidden="true" />
                  </Link>
                )}
              </div>
            )}

            {!isLoading &&
              !isError &&
              greenhouseFlowers.map((flower, index) => {
                const completedPetals = completedPetalsFor(flower);
                const completeness = completenessFor(flower);
                const isSelected = selectedFlower?.id === flower.id;
                return (
                  <article
                    key={flower.id}
                    data-bed-flower
                    className={clsx(
                      s.flowerPlot,
                      isSelected && s.flowerPlotActive,
                    )}
                    style={
                      {
                        "--plot-shift": `${plotDepths[index % plotDepths.length].shift}px`,
                        "--plot-scale":
                          plotDepths[index % plotDepths.length].scale,
                      } as CSSProperties
                    }
                  >
                    <div className={s.flowerStage}>
                      <KnowledgeFlowerGlyph
                        title={flower.title}
                        centerLabel={completeness === 1 ? "盛开" : "发芽"}
                        completedPetals={completedPetals}
                        selectedPetalId={
                          isSelected ? selectedPetalId : undefined
                        }
                        onPetalSelect={(petalId) => {
                          setSelectedFlowerId(flower.id);
                          setSelectedPetalId(petalId);
                        }}
                        size="small"
                        showLabels={false}
                        planted
                      />
                    </div>
                    <button
                      type="button"
                      className={s.plotSelect}
                      onClick={() => selectFlower(flower)}
                      aria-pressed={isSelected}
                      aria-label={`查看${flower.title}知识花详情`}
                    >
                      <span className={s.plotTitle}>{flower.title}</span>
                      <span className={s.plotMeta}>
                        {completedPetals.length} / {flowerPetals.length} 片花瓣
                        <span aria-hidden="true">·</span>
                        {formatFlowerDate(flower.plantedAt)}种植
                      </span>
                    </button>
                    <span className={s.plotSoil} aria-hidden="true" />
                  </article>
                );
              })}
          </div>
        </section>

        {selectedFlower ? (
          <aside
            className={s.panel}
            data-greenhouse-panel
            aria-labelledby="flower-detail-title"
          >
            <div className={s.panelHeader}>
              <div>
                <p className={s.panelKicker}>
                  观察手册 · {formatFlowerDate(selectedFlower.plantedAt)}种植
                </p>
                <h2 className={s.panelTitle} id="flower-detail-title">
                  {selectedFlower.title}
                </h2>
              </div>
              <span className={s.growthBadge}>
                <CheckCircledIcon aria-hidden="true" /> 已收藏
              </span>
            </div>
            <p className={s.panelMood}>
              选择花瓣，查看这项知识与旧经验、生活和行动的连接。
            </p>

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
                    style={{ "--petal-color": petal.color } as CSSProperties}
                    onClick={() => setSelectedPetalId(petal.id)}
                    aria-pressed={selectedPetal.id === petal.id}
                  >
                    <span className={s.petalSwatch} aria-hidden="true" />
                    {petal.label}
                  </button>
                );
              })}
            </div>

            <div className={s.detail} data-greenhouse-detail>
              <div className={s.detailHeading}>
                <div
                  className={s.detailMark}
                  style={
                    { "--petal-color": selectedPetal.color } as CSSProperties
                  }
                >
                  {selectedPetal.label}
                </div>
                <div>
                  <p className={s.detailLabel}>这片花瓣记录</p>
                  <h3 className={s.detailTitle}>{selectedPetal.short}</h3>
                </div>
              </div>
              <p className={s.detailBody}>
                {selectedFlower.petalContent[selectedPetal.id] ||
                  "这片花瓣还没有内容，回到学习单元继续培育吧。"}
              </p>
            </div>

            <div className={s.panelActions}>
              <Link
                to={`/project/${selectedFlower.projectSlug}/Extend`}
                className={s.editLink}
              >
                回到学习单元编辑 <ArrowRightIcon aria-hidden="true" />
              </Link>
              <p>温室用于收藏和回顾。内容修改仍在对应学习单元中完成。</p>
            </div>
          </aside>
        ) : (
          <aside className={clsx(s.panel, s.emptyPanel)} data-greenhouse-panel>
            <div className={s.emptyPanelMark} aria-hidden="true">
              <span />
            </div>
            <p className={s.panelKicker}>观察手册</p>
            <h2 className={s.panelTitle}>种下花朵后，这里会展示知识连接</h2>
            <p className={s.panelMood}>
              五片花瓣会帮你检查：我能把它接回旧知识吗？能在生活中看见吗？知道边界、行动和创作方向吗？
            </p>
          </aside>
        )}
      </div>
    </div>
  );
}
