import {describe, expect, it} from 'vitest';
import {model} from '../../wailsjs/go/models';
import {
    buildColumnUpdates,
    draftToColumnUpdate,
    fieldDraft,
    hasDraftChanges,
    hasFieldDraftChange,
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
