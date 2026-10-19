import {cellDetailText, isTruncatedBlob} from './cellDetail';
import {model} from '../api';

/** 'hex' sends the field as BLOB bytes written in hex; '' lets the column's type affinity apply. */
export type FieldEncoding = '' | 'hex';

export interface PlainColumnUpdate {
    column: string;
    text: string;
    isNull: boolean;
    encoding: FieldEncoding;
}

export function fieldDraft(cell: model.CellValue): string {
    if (cell.kind === 'null') {
        return '';
    }
    return cellDetailText(cell);
}

/**
 * BLOB cells are edited as hex. So is a NULL cell in a BLOB column. The encoding comes from
 * the cell and column, never from what the user typed, so text such as "0xcafe" stays text.
 */
export function fieldEncoding(cell: model.CellValue, meta?: model.ColumnInfo): FieldEncoding {
    if (cell.kind === 'blob') {
        return 'hex';
    }
    if (cell.kind === 'null' && (meta?.type ?? '').toUpperCase().includes('BLOB')) {
        return 'hex';
    }
    return '';
}

/** A field can be edited unless it only shows a preview of a larger BLOB. */
export function isFieldEditable(cell: model.CellValue): boolean {
    return !isTruncatedBlob(cell);
}

/** Builds column updates for changed fields only. */
export function buildColumnUpdates(
    columns: model.ColumnResult[],
    cells: model.CellValue[],
    drafts: Record<string, string>,
    columnMeta: model.ColumnInfo[],
): PlainColumnUpdate[] {
    const metaByName = new Map(columnMeta.map((c) => [c.name, c]));
    const updates: PlainColumnUpdate[] = [];

    columns.forEach((col, i) => {
        const meta = metaByName.get(col.name);
        if (!meta || !isFieldEditable(cells[i])) {
            return;
        }
        const original = fieldDraft(cells[i]);
        const draft = drafts[col.name] ?? '';
        if (draft === original) {
            return;
        }
        updates.push(draftToColumnUpdate(col.name, draft, cells[i], meta));
    });

    return updates;
}

export function draftToColumnUpdate(
    column: string,
    draft: string,
    cell: model.CellValue,
    meta: model.ColumnInfo,
): PlainColumnUpdate {
    if (!isFieldEditable(cell)) {
        throw new Error(`Column "${column}" holds a BLOB too large to edit here.`);
    }
    const encoding = fieldEncoding(cell, meta);
    if (draft === '') {
        if (!meta.notNull) {
            return {column, text: '', isNull: true, encoding};
        }
        const type = (meta.type ?? '').toUpperCase();
        if (encoding === 'hex' || type.includes('TEXT') || cell.kind === 'text') {
            return {column, text: '', isNull: false, encoding};
        }
        throw new Error(`Column "${column}" cannot be empty.`);
    }
    return {column, text: draft, isNull: false, encoding};
}

export function hasDraftChanges(
    columns: model.ColumnResult[],
    cells: model.CellValue[],
    drafts: Record<string, string>,
): boolean {
    return columns.some((col, i) => (drafts[col.name] ?? '') !== fieldDraft(cells[i]));
}

export function hasFieldDraftChange(draft: string, cell: model.CellValue): boolean {
    return draft !== fieldDraft(cell);
}
