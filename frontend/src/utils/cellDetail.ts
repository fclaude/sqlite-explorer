import {formatCellDisplay, model} from '../api';

export interface RecordRowContext {
    source: 'table' | 'query';
    tableName?: string;
    rowIndex: number;
    page?: number;
    rowId?: string;
    editable?: boolean;
    columnMeta?: model.ColumnInfo[];
    columns: model.ColumnResult[];
    cells: model.CellValue[];
}

export interface CellDetailContext extends RecordRowContext {
    columnIndex: number;
    columnName: string;
    cell: model.CellValue;
}

export function rowLabel(ctx: Pick<RecordRowContext, 'rowIndex' | 'page'>): string {
    if (ctx.page != null && ctx.page > 0) {
        return `Row ${ctx.rowIndex + 1} (page ${ctx.page})`;
    }
    return `Row ${ctx.rowIndex + 1}`;
}

/** Full text shown in the detail editor (not grid-truncated display). */
export function cellDetailText(cell: model.CellValue): string {
    switch (cell.kind) {
        case 'null':
            return '';
        case 'blob': {
            const v = cell.value as {hex?: string; size?: number};
            const size = v?.size ?? 0;
            const hex = v?.hex ?? '';
            const previewNote =
                size > (hex.length / 2)
                    ? `\n\n[Preview: first ${hex.length / 2} of ${size} bytes. BLOB export or SQL may be needed for the full value.]`
                    : '';
            return hex ? `0x${hex}${previewNote}` : `[BLOB ${size} bytes]`;
        }
        case 'text':
        case 'int':
        case 'real':
            return String(cell.value ?? '');
        default:
            return String(cell.value ?? '');
    }
}

export function cellTypeLabel(cell: model.CellValue): string {
    switch (cell.kind) {
        case 'null':
            return 'NULL';
        case 'blob':
            return 'BLOB';
        case 'int':
            return 'INTEGER';
        case 'real':
            return 'REAL';
        case 'text':
            return 'TEXT';
        default:
            return cell.kind.toUpperCase();
    }
}

export function isCellExpandable(cell: model.CellValue): boolean {
    if (cell.kind === 'null') {
        return true;
    }
    const display = formatCellDisplay(cell);
    return display.length > 48 || cell.kind === 'blob';
}
