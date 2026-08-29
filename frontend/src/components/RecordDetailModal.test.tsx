import {render, screen, fireEvent, waitFor} from '@testing-library/react';
import {describe, expect, it, vi} from 'vitest';
import {RecordDetailModal} from './RecordDetailModal';
import {RecordRowContext} from '../utils/cellDetail';
import {model} from '../../wailsjs/go/models';

const baseContext: RecordRowContext = {
    source: 'table',
    tableName: 'customers',
    rowIndex: 0,
    page: 1,
    rowId: '42',
    editable: true,
    columnMeta: [
        new model.ColumnInfo({name: 'id', type: 'INTEGER', notNull: true, primaryKey: 1}),
        new model.ColumnInfo({name: 'name', type: 'TEXT', notNull: true, primaryKey: 0}),
    ],
    columns: [
        {name: 'id', type: 'INTEGER'},
        {name: 'name', type: 'TEXT'},
    ],
    cells: [
        {kind: 'int', value: 1},
        {kind: 'text', value: 'Ada'},
    ],
};

describe('RecordDetailModal', () => {
    it('shows Save when editable and calls onSave with drafts', async () => {
        const onSave = vi.fn().mockResolvedValue(undefined);
        const onClose = vi.fn();

        render(
            <RecordDetailModal
                context={baseContext}
                onClose={onClose}
                onSave={onSave}
            />,
        );

        const nameField = screen.getByLabelText('name');
        fireEvent.change(nameField, {target: {value: 'Example Customer 001'}});

        const saveBtn = screen.getByRole('button', {name: 'Save'});
        expect(saveBtn).not.toBeDisabled();
        fireEvent.click(saveBtn);

        await waitFor(() => {
            expect(onSave).toHaveBeenCalledWith(
                expect.objectContaining({name: 'Example Customer 001'}),
            );
            expect(onClose).toHaveBeenCalled();
        });
    });

    it('hides Save for query results', () => {
        render(
            <RecordDetailModal
                context={{...baseContext, source: 'query', editable: false}}
                onClose={() => {}}
                onSave={vi.fn()}
            />,
        );
        expect(screen.queryByRole('button', {name: 'Save'})).toBeNull();
        expect(screen.getByText(/read-only/i)).toBeInTheDocument();
    });
});
