import {cellDetailText} from './cellDetail';
import {model} from '../api';

export interface PlainColumnUpdate {
    column: string;
    text: string;
    isNull: boolean;
}

export function fieldDraft(cell: model.CellValue): string {
    if (cell.kind === 'null') {
        return '';
    }
    return cellDetailText(cell);
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
        if (!meta) {
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
    const trimmed = draft;
    if (trimmed === '') {
        if (!meta.notNull) {
            return {column, text: '', isNull: true};
        }
        const type = (meta.type ?? '').toUpperCase();
        if (type.includes('TEXT') || cell.kind === 'text') {
            return {column, text: '', isNull: false};
        }
        throw new Error(`Column "${column}" cannot be empty.`);
    }
    return {column, text: trimmed, isNull: false};
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
