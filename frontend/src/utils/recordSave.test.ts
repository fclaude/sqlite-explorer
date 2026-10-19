import {describe, expect, it} from 'vitest';
import {model} from '../../wailsjs/go/models';
import {
    buildColumnUpdates,
    draftToColumnUpdate,
    fieldDraft,
    fieldEncoding,
    hasDraftChanges,
    hasFieldDraftChange,
    isFieldEditable,
} from './recordSave';

describe('buildColumnUpdates', () => {
    const columns: model.ColumnResult[] = [
        {name: 'id', type: 'INTEGER'},
        {name: 'name', type: 'TEXT'},
    ];
    const cells: model.CellValue[] = [
        {kind: 'int', value: 1},
        {kind: 'text', value: 'Ada'},
    ];
    const meta: model.ColumnInfo[] = [
        {name: 'id', type: 'INTEGER', notNull: true, primaryKey: 1},
        {name: 'name', type: 'TEXT', notNull: true, primaryKey: 0},
    ];

    it('includes only changed columns', () => {
        const updates = buildColumnUpdates(columns, cells, {id: '1', name: 'Example Customer 001'}, meta);
        expect(updates).toHaveLength(1);
        expect(updates[0].column).toBe('name');
        expect(updates[0].text).toBe('Example Customer 001');
    });

    it('maps empty nullable field to null', () => {
        const nullableMeta: model.ColumnInfo[] = [
            {name: 'body', type: 'TEXT', notNull: false, primaryKey: 0},
        ];
        const cols: model.ColumnResult[] = [{name: 'body', type: 'TEXT'}];
        const row: model.CellValue[] = [{kind: 'null', value: null}];
        const updates = buildColumnUpdates(cols, row, {body: 'hello'}, nullableMeta);
        expect(updates[0].isNull).toBe(false);
        expect(updates[0].text).toBe('hello');

        const nullUpdates = buildColumnUpdates(cols, [{kind: 'text', value: 'x'}], {body: ''}, nullableMeta);
        expect(nullUpdates[0].isNull).toBe(true);
    });
});

describe('draftToColumnUpdate', () => {
    it('rejects empty required numeric fields', () => {
        const meta = new model.ColumnInfo({name: 'id', type: 'INTEGER', notNull: true});
        expect(() =>
            draftToColumnUpdate('id', '', {kind: 'int', value: 1}, meta),
        ).toThrow(/cannot be empty/);
    });
});

describe('hasDraftChanges', () => {
    it('detects modifications', () => {
        const cols = [{name: 'n', type: 'TEXT'}];
        const cells: model.CellValue[] = [{kind: 'text', value: 'a'}];
        expect(hasDraftChanges(cols, cells, {n: 'a'})).toBe(false);
        expect(hasDraftChanges(cols, cells, {n: 'b'})).toBe(true);
    });
});

describe('fieldDraft', () => {
    it('matches cell text', () => {
        expect(fieldDraft({kind: 'text', value: 'hi'})).toBe('hi');
    });
});

describe('hasFieldDraftChange', () => {
    it('detects edits', () => {
        const cell: model.CellValue = {kind: 'text', value: 'a'};
        expect(hasFieldDraftChange('a', cell)).toBe(false);
        expect(hasFieldDraftChange('b', cell)).toBe(true);
    });
});

describe('field encoding', () => {
    const textMeta = new model.ColumnInfo({name: 'name', type: 'TEXT', notNull: false});
    const blobMeta = new model.ColumnInfo({name: 'data', type: 'BLOB', notNull: false});

    it('never treats typed text as hex', () => {
        const update = draftToColumnUpdate('name', '0xcafe', {kind: 'text', value: 'a'}, textMeta);
        expect(update).toEqual({column: 'name', text: '0xcafe', isNull: false, encoding: ''});
    });

    it('edits BLOB cells and NULL cells of BLOB columns as hex', () => {
        const blob: model.CellValue = {kind: 'blob', value: {hex: '00ff', size: 2}};
        expect(fieldEncoding(blob, textMeta)).toBe('hex');
        expect(fieldEncoding({kind: 'null', value: null}, blobMeta)).toBe('hex');
        expect(fieldEncoding({kind: 'null', value: null}, textMeta)).toBe('');
        expect(draftToColumnUpdate('data', '0x0102', blob, blobMeta).encoding).toBe('hex');
    });

    it('sends an empty BLOB for a cleared required hex field', () => {
        const meta = new model.ColumnInfo({name: 'data', type: 'BLOB', notNull: true});
        const blob: model.CellValue = {kind: 'blob', value: {hex: '00', size: 1}};
        expect(draftToColumnUpdate('data', '', blob, meta)).toEqual({
            column: 'data', text: '', isNull: false, encoding: 'hex',
        });
    });

    it('refuses to save a truncated BLOB preview', () => {
        const preview: model.CellValue = {kind: 'blob', value: {hex: 'ab'.repeat(64), size: 200}};
        expect(isFieldEditable(preview)).toBe(false);
        expect(() => draftToColumnUpdate('data', '0x00', preview, blobMeta)).toThrow(/too large/);
        const updates = buildColumnUpdates(
            [{name: 'data', type: 'BLOB'}],
            [preview],
            {data: '0x00'},
            [blobMeta],
        );
        expect(updates).toEqual([]);
    });
});
