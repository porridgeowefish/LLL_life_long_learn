import clsx from 'clsx';

import { ALL_ZONES, type ZoneName, ZONE_DISPLAY } from '@/shared/types/domain';
import { Icon, type IconName } from '@/shared/primitive/Icon';

import s from './ZoneTimeline.module.css';

const ZONE_ICON: Record<ZoneName, IconName> = {
  Intro: 'leaf',
  Explain: 'book',
  Practice: 'pen',
  Extend: 'link',
  Summary: 'file',
};

interface ZoneTimelineProps {
  currentZone: ZoneName | null;
  completedZones?: Set<ZoneName>;
  hasOutput?: Set<ZoneName>; // zones with a non-empty output file
  onSelect?: (z: ZoneName) => void;
}

export function ZoneTimeline({
  currentZone,
  hasOutput,
  onSelect,
}: ZoneTimelineProps) {
  return (
    <ol className={s.timeline}>
      {ALL_ZONES.map((zone, i) => {
        const done = hasOutput?.has(zone) ?? false;
        const current = currentZone === zone;
        return (
          <li
            key={zone}
            className={clsx(
              s.item,
              done && s.done,
              current && s.current,
              !done && !current && s.pending,
            )}
          >
            <button
              type="button"
              className={s.row}
              onClick={() => onSelect?.(zone)}
            >
              <span className={s.dot}>
                {done ? <Icon name="check" size={11} /> : <span className={s.dotInner} />}
              </span>
              <span className={s.body}>
                <Icon name={ZONE_ICON[zone]} size={14} className={s.zoneIcon} />
                <span className={s.label}>{ZONE_DISPLAY[zone]}</span>
              </span>
            </button>
            {i < ALL_ZONES.length - 1 && <span className={s.line} />}
          </li>
        );
      })}
    </ol>
  );
}
