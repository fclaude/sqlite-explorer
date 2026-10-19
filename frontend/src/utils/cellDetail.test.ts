import {describe, expect, it} from 'vitest';
import {model} from '../api';
import {cellDetailText, isCellExpandable, isTruncatedBlob} from './cellDetail';

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

describe('isTruncatedBlob', () => {
    it('detects preview-only BLOBs', () => {
        expect(isTruncatedBlob({kind: 'blob', value: {hex: 'ab'.repeat(64), size: 200}})).toBe(true);
        expect(isTruncatedBlob({kind: 'blob', value: {hex: 'abcd', size: 2}})).toBe(false);
        expect(isTruncatedBlob({kind: 'text', value: 'x'})).toBe(false);
    });
});
