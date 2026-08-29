import {
    CloseDatabase,
    DatabaseInfo,
    GetSchema,
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

    runQuery(req: model.QueryRequest): Promise<model.QueryResponse> {
        return RunQuery(req);
    },

    cancelQuery(queryId: number): Promise<void> {
        return CancelQuery(queryId);
    },

    exportTablePage(tableRows: model.TableRowsRequest): Promise<void> {
        const req = new model.ExportRequest({
            source: 'tablePage',
            tableRows: tableRows,
        });
        return ExportRowsToCSV(req);
    },

    updateTableRow(params: {
        table: string;
        rowId: string;
        updates: Array<{column: string; text: string; isNull: boolean}>;
    }): Promise<model.UpdateTableRowResponse> {
        return UpdateTableRow(
            new model.UpdateTableRowRequest({
                table: params.table,
                rowId: params.rowId,
                updates: params.updates.map((u) => new model.ColumnUpdate(u)),
            }),
        );
    },

    exportQueryResult(sql: string): Promise<void> {
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

/** Maps Wails/Go errors to user-facing text (no stack traces). */
export function formatAPIError(err: unknown): {message: string; detail: string} {
    if (err instanceof Error) {
        const message = err.message.trim() || 'An unexpected error occurred.';
        return {message, detail: ''};
    }
    const message = String(err).trim() || 'An unexpected error occurred.';
    return {message, detail: ''};
}
