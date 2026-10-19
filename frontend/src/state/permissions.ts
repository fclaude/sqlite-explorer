const STORAGE_KEY = 'sqlite-explorer.sqlPermissions';

/** Statement categories the user allowed in the SQL editor, beyond read queries. */
export function loadAllowedCategories(): string[] {
    try {
        const parsed: unknown = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '[]');
        return Array.isArray(parsed) ? parsed.filter((id): id is string => typeof id === 'string') : [];
    } catch {
        return [];
    }
}

export function saveAllowedCategories(ids: string[]): void {
    try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(ids));
    } catch {
        // Storage can be unavailable; the choice then lasts for this session only.
    }
}
