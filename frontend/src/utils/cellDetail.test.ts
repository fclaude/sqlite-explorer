import {describe, expect, it} from 'vitest';
import {model} from '../api';
import {cellDetailText, isCellExpandable} from './cellDetail';

describe('cellDetailText', () => {
    it('returns full text for long strings', () => {
        const long = 'a'.repeat(500);
        const cell: model.CellValue = {kind: 'text', value: long};
        expect(cellDetailText(cell)).toBe(long);
    });

    it('notes blob preview limit', () => {
        const hex = 'ab'.repeat(64);
        const cell: model.CellValue = {
            kind: 'blob',
            value: {hex, size: 200},
        };
        expect(cellDetailText(cell)).toContain('Preview');
        expect(cellDetailText(cell)).toContain('200');
    });
});

describe('isCellExpandable', () => {
    it('marks long text expandable', () => {
        const cell: model.CellValue = {kind: 'text', value: 'x'.repeat(60)};
        expect(isCellExpandable(cell)).toBe(true);
    });
});
