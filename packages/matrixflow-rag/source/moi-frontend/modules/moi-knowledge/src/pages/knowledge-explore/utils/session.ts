import type { TFunction } from 'i18next';

import { type SessionRecord } from '../../../service/dialogSession';

export interface SessionGroup {
  key: string;
  label: string;
  timestamp: number;
  sessions: SessionRecord[];
}

function getSafeTimestamp(input: number | undefined): number {
  if (typeof input === 'number' && Number.isFinite(input) && input > 0) {
    return input;
  }
  return Math.floor(Date.now() / 1000);
}

function isSameDay(left: Date, right: Date): boolean {
  return left.getFullYear() === right.getFullYear() && left.getMonth() === right.getMonth() && left.getDate() === right.getDate();
}

function createMonthLabel(locale: string, timestampMs: number): string {
  const formatter = new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'long',
  });
  return formatter.format(new Date(timestampMs));
}

export function groupSessionsByMonth(sessions: SessionRecord[], locale: string, t: TFunction<'moi-knowledge'>): SessionGroup[] {
  if (!Array.isArray(sessions) || sessions.length === 0) {
    return [];
  }

  const today = new Date();
  const todayGroup: SessionGroup = {
    key: 'today',
    label: t('knowledge.explore.group-today'),
    timestamp: today.getTime(),
    sessions: [],
  };
  const monthGroupMap = new Map<string, SessionGroup>();

  sessions.forEach((session) => {
    const createdAt = getSafeTimestamp(session.created_at);
    const createdDate = new Date(createdAt * 1000);

    if (isSameDay(createdDate, today)) {
      todayGroup.sessions.push(session);
      return;
    }

    const monthStart = new Date(createdDate.getFullYear(), createdDate.getMonth(), 1, 0, 0, 0, 0);
    const monthKey = `${monthStart.getFullYear()}-${monthStart.getMonth() + 1}`;
    const monthTimestamp = monthStart.getTime();

    if (!monthGroupMap.has(monthKey)) {
      monthGroupMap.set(monthKey, {
        key: monthKey,
        label: createMonthLabel(locale, monthTimestamp),
        timestamp: monthTimestamp,
        sessions: [],
      });
    }

    monthGroupMap.get(monthKey)?.sessions.push(session);
  });

  const groups: SessionGroup[] = [];
  if (todayGroup.sessions.length > 0) {
    groups.push(todayGroup);
  }

  groups.push(...Array.from(monthGroupMap.values()).sort((left, right) => right.timestamp - left.timestamp));

  return groups;
}

export function withPinnedTag(session: SessionRecord, pinned: boolean): SessionRecord {
  const baseTags = (session.tags || []).filter((tag) => {
    const name = typeof tag === 'string' ? tag : tag.name;
    return name !== 'pinned' && name !== 'unpinned';
  });

  return {
    ...session,
    tags: [
      ...baseTags,
      {
        name: pinned ? 'pinned' : 'unpinned',
        source: 'moi',
        created_at: Math.floor(Date.now() / 1000),
        updated_at: Math.floor(Date.now() / 1000),
      },
    ],
  };
}
