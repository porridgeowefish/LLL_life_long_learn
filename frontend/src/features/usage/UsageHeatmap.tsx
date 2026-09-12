import { useMemo, useState } from 'react';

import { useActivity } from '@/features/projects';

import s from './UsagePage.module.css';

// A compact learning-investment calendar for the usage page — the same heat
// scale as the homepage rhythm section, without the streak/detail panels.
export function UsageHeatmap() {
  const [weeks, setWeeks] = useState<26 | 52>(26);
  const activity = useActivity(weeks);
  const summary = activity.data;
  const [selected, setSelected] = useState('');

  const monthLabels = useMemo(() => {
    const days = summary?.days ?? [];
    const labels: string[] = [];
    let previous = -1;
    for (let i = 0; i < days.length; i += 7) {
      const month = Number(days[i]?.date.slice(5, 7) ?? 0);
      labels.push(month && month !== previous ? `${month}月` : '');
      previous = month || previous;
    }
    return labels;
  }, [summary?.days]);

  const selectedDay = (summary?.days ?? []).find((day) => day.date === selected);

  return (
    <section className={s.heatSection} aria-labelledby="usage-heat-title">
      <header className={s.heatHead}>
        <div>
          <span>学习投入</span>
          <h2 id="usage-heat-title">投入热力图</h2>
          <p>与首页学习节律同一数据源：颜色表示每日学习投入，便于对照 Token 用量看活跃分布。</p>
        </div>
        <div className={s.heatFilters}>
          <button type="button" className={weeks === 26 ? s.heatActive : ''} onClick={() => setWeeks(26)}>近26周</button>
          <button type="button" className={weeks === 52 ? s.heatActive : ''} onClick={() => setWeeks(52)}>近一年</button>
        </div>
      </header>
      <div className={s.heatScroll}>
        <div className={s.heatMonths} style={{ gridTemplateColumns: `repeat(${monthLabels.length}, 15px)` }}>
          {monthLabels.map((label, index) => <span key={`${label}-${index}`}>{label}</span>)}
        </div>
        <div className={s.heatLine}>
          <div className={s.heatWeekdays} aria-hidden="true"><span>一</span><span>三</span><span>五</span></div>
          <div className={s.heatGrid} role="grid" aria-label={`最近${weeks}周学习投入`}>
            {(summary?.days ?? []).map((day) => (
              <button
                key={day.date}
                type="button"
                role="gridcell"
                className={`${s.heatDay} ${selected === day.date ? s.heatSelected : ''}`}
                data-level={heatLevel(day.activity)}
                title={`${day.date} · 热度 ${day.activity} · ${day.actions} 次学习行动`}
                aria-label={`${day.date}，投入热度 ${day.activity}，${day.actions} 次学习行动`}
                onClick={() => setSelected(day.date)}
              />
            ))}
          </div>
        </div>
        <div className={s.heatLegend}>
          <span>{selectedDay ? `${selectedDay.date} · 热度 ${selectedDay.activity} · ${selectedDay.actions} 次行动` : `近${weeks}周活跃 ${summary?.activeDays ?? 0} 天 · 有效行动 ${summary?.totalActions ?? 0} 次`}</span>
          <span>少 <i data-level="0" /><i data-level="1" /><i data-level="2" /><i data-level="3" /><i data-level="4" /> 多</span>
        </div>
      </div>
    </section>
  );
}

function heatLevel(value: number) { return value === 0 ? 0 : value <= 2 ? 1 : value <= 5 ? 2 : value <= 9 ? 3 : 4; }
