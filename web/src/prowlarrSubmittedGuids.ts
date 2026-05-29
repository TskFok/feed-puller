export const PROWLARR_SUBMITTED_STORAGE_KEY = 'feed-puller-prowlarr-submitted-guids';

export function readSessionSubmittedGuids(): Set<string> {
  if (typeof sessionStorage === 'undefined') {
    return new Set();
  }
  try {
    const raw = sessionStorage.getItem(PROWLARR_SUBMITTED_STORAGE_KEY);
    if (!raw) return new Set();
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return new Set();
    return new Set(parsed.filter((value): value is string => typeof value === 'string' && value.length > 0));
  } catch {
    return new Set();
  }
}

export function addSessionSubmittedGuids(guids: Iterable<string>): void {
  if (typeof sessionStorage === 'undefined') {
    return;
  }
  const next = readSessionSubmittedGuids();
  for (const guid of guids) {
    const trimmed = guid.trim();
    if (trimmed) next.add(trimmed);
  }
  sessionStorage.setItem(PROWLARR_SUBMITTED_STORAGE_KEY, JSON.stringify([...next]));
}

export function mergeSubmittedGuids(
  backendGuids: Iterable<string>,
  sessionGuids: ReadonlySet<string>,
  resultGuids: Iterable<string>
): Set<string> {
  const allowed = new Set(resultGuids);
  const merged = new Set<string>();
  for (const guid of backendGuids) {
    if (allowed.has(guid)) merged.add(guid);
  }
  for (const guid of sessionGuids) {
    if (allowed.has(guid)) merged.add(guid);
  }
  return merged;
}
