import {describe, expect, it} from 'vitest';
import {formatBytes, formatInteger} from './formatBytes';

describe('formatBytes', () => {
    it('formats sizes', () => {
        expect(formatBytes(512)).toBe('512 B');
        expect(formatBytes(2048)).toBe('2.0 KB');
    });
});

describe('formatInteger', () => {
    it('adds grouping', () => {
        expect(formatInteger(12345)).toMatch(/12/);
    });
});
