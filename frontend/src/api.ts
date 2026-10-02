import {
    CloseDatabase,
    DatabaseInfo,
    GetSchema,
    GetStatementCategories,
    GetTableRows,
    GetObjectStats,
    OpenDatabase,
    ExportRowsToCSV,
    RunQuery,
    CancelQuery,
    UpdateTableRow,
} from '../wailsjs/go/backend/App';
import {model} from '../wailsjs/go/models';
/** Typed wrapper over Wails-generated bindings. Components must use this, not wailsjs directly. */
export const WailsAPI = {
    openDatabase(): Promise<model.DatabaseInfo> {
        return OpenDatabase();
    },

    closeDatabase(): Promise<void> {
        return CloseDatabase();
    },

    databaseInfo(): Promise<model.DatabaseInfo> {
        return DatabaseInfo();
    },

    getSchema(): Promise<model.SchemaInfo> {
        return GetSchema();
    },

    getTableRows(req: model.TableRowsRequest): Promise<model.TableRowsResponse> {
        return GetTableRows(req);
    },

    getObjectStats(name: string): Promise<model.ObjectStats> {
        return GetObjectStats(name);
    },

    runQuery(req: {sql: string; allow: string[]}): Promise<model.QueryResponse> {
        return RunQuery(new model.QueryRequest(req));
    },

    cancelQuery(queryId: number): Promise<void> {
        return CancelQuery(queryId);
    },

    getStatementCategories(): Promise<model.StatementCategory[]> {
        return GetStatementCategories();
    },

    /** Exports the current page, or with scope 'all' every row matching the filter and sort. */
    exportTable(tableRows: model.TableRowsRequest, scope: 'page' | 'all'): Promise<model.ExportResult> {
        const req = new model.ExportRequest({
            source: scope === 'all' ? 'table' : 'tablePage',
            tableRows: tableRows,
        });
        return ExportRowsToCSV(req);
    },

    updateTableRow(params: {
        table: string;
        rowId: string;
        updates: Array<{column: string; text: string; isNull: boolean; encoding: string}>;
    }): Promise<model.UpdateTableRowResponse> {
        return UpdateTableRow(
            new model.UpdateTableRowRequest({
                table: params.table,
                rowId: params.rowId,
                updates: params.updates.map((u) => new model.ColumnUpdate(u)),
            }),
        );
    },

    /** Runs a read-only query again and exports all of its rows. */
    exportQueryResult(sql: string): Promise<model.ExportResult> {
        const req = new model.ExportRequest({
            source: 'queryResult',
            sql: sql,
        });
        return ExportRowsToCSV(req);
    },
};

export function formatCellDisplay(cell: model.CellValue): string {
    switch (cell.kind) {
        case 'null':
            return 'NULL';
        case 'blob': {
            const v = cell.value as {hex?: string; size?: number};
            if (v && typeof v.size === 'number') {
                return `<BLOB ${v.size} bytes>`;
            }
            return '<BLOB>';
        }
        case 'text':
            return String(cell.value ?? '');
        case 'int':
        case 'real':
            return String(cell.value ?? '');
        default:
            return String(cell.value ?? '');
    }
}

export function blobCellTooltip(cell: model.CellValue): string | undefined {
    if (cell.kind !== 'blob') {
        return undefined;
    }
    const v = cell.value as {hex?: string; size?: number};
    if (!v?.hex) {
        return undefined;
    }
    return `0x${v.hex}${(v.size ?? 0) > 64 ? '...' : ''}`;
}

export type {model};

export interface APIError {
    code: string;
    message: string;
    detail: string;
}

/**
 * Maps Wails/Go errors to user-facing text. The backend sends application errors as a JSON
 * string ({code, message, detail}) because the Wails runtime only preserves string errors.
 */
export function formatAPIError(err: unknown): APIError {
    const raw = (err instanceof Error ? err.message : String(err ?? '')).trim();
    if (raw.startsWith('{')) {
        try {
            const parsed = JSON.parse(raw) as Partial<APIError>;
            if (typeof parsed.message === 'string') {
                const message = parsed.message.trim() || 'An unexpected error occurred.';
                const detail = typeof parsed.detail === 'string' && parsed.detail !== message ? parsed.detail : '';
                return {code: typeof parsed.code === 'string' ? parsed.code : '', message, detail};
            }
        } catch {
            // Not a structured error; show it as text.
        }
    }
    return {code: '', message: raw || 'An unexpected error occurred.', detail: ''};
}
