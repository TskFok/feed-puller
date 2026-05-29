import { describe, expect, it, beforeEach } from 'vitest';
import {
  PROWLARR_SUBMITTED_STORAGE_KEY,
  addSessionSubmittedGuids,
  mergeSubmittedGuids,
  readSessionSubmittedGuids
} from './prowlarrSubmittedGuids';

describe('prowlarrSubmittedGuids', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it('readSessionSubmittedGuids 从 sessionStorage 恢复', () => {
    sessionStorage.setItem(PROWLARR_SUBMITTED_STORAGE_KEY, JSON.stringify(['g1', 'g2']));
    expect([...readSessionSubmittedGuids()]).toEqual(['g1', 'g2']);
  });

  it('addSessionSubmittedGuids 会合并写入', () => {
    addSessionSubmittedGuids(['g1']);
    addSessionSubmittedGuids(['g2', 'g1']);
    expect([...readSessionSubmittedGuids()].sort()).toEqual(['g1', 'g2']);
  });

  it('mergeSubmittedGuids 仅保留当前结果中的 guid', () => {
    const merged = mergeSubmittedGuids(['g1', 'g9'], new Set(['g2', 'g9']), ['g1', 'g2', 'g3']);
    expect([...merged].sort()).toEqual(['g1', 'g2']);
  });
});
