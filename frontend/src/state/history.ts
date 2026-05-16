const MAX_HISTORY = 50;

let queryHistory: string[] = [];

export function addQueryHistory(sql: string): void {
    const trimmed = sql.trim();
    if (!trimmed) {
        return;
    }
    queryHistory = queryHistory.filter((q) => q !== trimmed);
    queryHistory.unshift(trimmed);
    if (queryHistory.length > MAX_HISTORY) {
        queryHistory = queryHistory.slice(0, MAX_HISTORY);
    }
}

export function getQueryHistory(): string[] {
    return [...queryHistory];
}

export function clearQueryHistory(): void {
    queryHistory = [];
}
