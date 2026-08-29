import {render, screen} from '@testing-library/react';
import {describe, expect, it, vi} from 'vitest';
import {TableStatsView} from './TableStatsView';

vi.mock('../api', () => ({
    WailsAPI: {
        getObjectStats: vi.fn().mockResolvedValue({
            name: 'customers',
            kind: 'table',
            rowCount: 5,
            columnCount: 3,
            indexCount: 0,
            foreignKeyCount: 0,
            triggerCount: 0,
            primaryKeyColumns: ['id'],
            withoutRowId: false,
            databaseFileBytes: 24576,
            databasePageCount: 6,
            databasePageSize: 4096,
            databaseUsedBytes: 24576,
            durationMs: 2,
        }),
    },
    formatAPIError: (e: unknown) => ({message: String(e), detail: ''}),
}));

describe('TableStatsView', () => {
    it('loads and shows row count', async () => {
        render(
            <TableStatsView
                selected={{kind: 'table', name: 'customers'}}
                hasDatabase={true}
                onOpenDatabase={() => {}}
            />,
        );
        expect(await screen.findByText('5')).toBeInTheDocument();
        expect(screen.getByText('Rows (exact)')).toBeInTheDocument();
    });

    it('prompts when nothing selected', () => {
        render(
            <TableStatsView selected={null} hasDatabase={true} onOpenDatabase={() => {}} />,
        );
        expect(screen.getByText(/Select a table or view/)).toBeInTheDocument();
    });
});
