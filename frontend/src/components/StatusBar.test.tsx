import {render, screen} from '@testing-library/react';
import {describe, expect, it, vi} from 'vitest';
import {initialState} from '../state/types';
import {StatusBar} from './StatusBar';

describe('StatusBar', () => {
    it('shows No database when no db is open', () => {
        render(
            <StatusBar dbInfo={null} status={initialState.status} onClearError={vi.fn()} />,
        );
        expect(screen.getByText('No database')).toBeInTheDocument();
    });

    it('shows database path when open', () => {
        render(
            <StatusBar
                dbInfo={{path: '/tmp/test.sqlite', sizeBytes: 1024, readOnly: true}}
                status={initialState.status}
                onClearError={vi.fn()}
            />,
        );
        expect(screen.getByText('/tmp/test.sqlite')).toBeInTheDocument();
    });
});
