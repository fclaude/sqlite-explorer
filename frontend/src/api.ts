import {
    CloseDatabase,
    DatabaseInfo,
    GetSchema,
    GetTableRows,
    OpenDatabase,
    RunQuery,
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

    runQuery(req: model.QueryRequest): Promise<model.QueryResponse> {
        return RunQuery(req);
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

export function formatAPIError(err: unknown): {message: string; detail: string} {
    if (err instanceof Error) {
        return {message: err.message, detail: err.stack ?? ''};
    }
    return {message: String(err), detail: ''};
}
