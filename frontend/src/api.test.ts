import {describe, expect, it} from 'vitest';
import {formatAPIError} from './api';

describe('formatAPIError', () => {
    it('decodes structured backend errors', () => {
        const err = new Error(JSON.stringify({
            code: 'MALFORMED_SQL',
            message: 'The SQL could not be parsed. Check your syntax.',
            detail: 'SQL logic error: near "FORM": syntax error (1)',
        }));
        expect(formatAPIError(err)).toEqual({
            code: 'MALFORMED_SQL',
            message: 'The SQL could not be parsed. Check your syntax.',
            detail: 'SQL logic error: near "FORM": syntax error (1)',
        });
    });

    it('drops detail that repeats the message', () => {
        const err = new Error(JSON.stringify({code: 'X', message: 'Same', detail: 'Same'}));
        expect(formatAPIError(err).detail).toBe('');
    });

    it('falls back to plain text', () => {
        expect(formatAPIError(new Error('boom'))).toEqual({code: '', message: 'boom', detail: ''});
        expect(formatAPIError('{not json')).toEqual({code: '', message: '{not json', detail: ''});
        expect(formatAPIError(undefined).message).toBe('An unexpected error occurred.');
    });
});
