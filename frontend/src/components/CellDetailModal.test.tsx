import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import {describe, expect, it, vi} from 'vitest';
import {CellDetailModal} from './CellDetailModal';
import {CellDetailContext} from '../utils/cellDetail';

const baseContext: CellDetailContext = {
    source: 'table',
    tableName: 'customers',
    rowIndex: 0,
    page: 1,
    rowId: '7',
    editable: true,
    columnMeta: [],
    columns: [{name: 'name', type: 'TEXT'}],
    cells: [{kind: 'text', value: 'Ada'}],
    columnIndex: 0,
    columnName: 'name',
    cell: {kind: 'text', value: 'Ada'},
};

describe('CellDetailModal', () => {
    it('saves a single field when Save is clicked', async () => {
        const onSave = vi.fn().mockResolvedValue(undefined);
        const onClose = vi.fn();

        render(
            <CellDetailModal context={baseContext} onClose={onClose} onSave={onSave} />,
        );

        fireEvent.change(screen.getByRole('textbox'), {target: {value: 'Example Customer 003'}});
        fireEvent.click(screen.getByRole('button', {name: 'Save'}));

        await waitFor(() => {
            expect(onSave).toHaveBeenCalledWith('Example Customer 003');
            expect(onClose).toHaveBeenCalled();
        });
    });

    it('does not show Save for read-only query cells', () => {
        render(
            <CellDetailModal
                context={{...baseContext, source: 'query', editable: false}}
                onClose={() => {}}
                onSave={vi.fn()}
            />,
        );
        expect(screen.queryByRole('button', {name: 'Save'})).toBeNull();
    });
});
