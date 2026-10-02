import {act, fireEvent, render, screen, waitFor} from '@testing-library/react';
import {beforeEach, describe, expect, it, vi} from 'vitest';
import {model} from '../../wailsjs/go/models';
import {AppProvider} from '../state/AppProvider';
import {DataGrid} from './DataGrid';

const api = vi.hoisted(() => ({
    getTableRows: vi.fn(),
    exportTable: vi.fn(),
    cancelQuery: vi.fn(),
}));

vi.mock('../api', async (importOriginal) => {
    const actual = await importOriginal<typeof import('../api')>();
    return {...actual, WailsAPI: api};
});

function rowsFor(req: model.TableRowsRequest, label = req.table): model.TableRowsResponse {
    return {
        columns: [{name: 'id', type: 'INTEGER'}, {name: 'name', type: 'TEXT'}],
        rows: [[{kind: 'int', value: req.page}, {kind: 'text', value: `${label} page ${req.page}`}]],
        editable: false,
        page: req.page,
        pageSize: req.pageSize,
        totalRows: 500,
        durationMs: 1,
    } as model.TableRowsResponse;
}

function renderGrid(name: string) {
    const ui = (n: string) => (
        <AppProvider>
            <DataGrid selected={{kind: 'table', name: n}} hasDatabase={true} onOpenDatabase={() => {}} />
        </AppProvider>
    );
    const result = render(ui(name));
    return {...result, select: (n: string) => result.rerender(ui(n))};
}

describe('DataGrid', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        api.getTableRows.mockImplementation(async (req: model.TableRowsRequest) => rowsFor(req));
    });

    it('fetches a newly selected table once, with a fresh view', async () => {
        const {select} = renderGrid('customers');
        await screen.findByText('customers page 1');

        fireEvent.click(screen.getByRole('button', {name: 'name'}));
        await waitFor(() => expect(api.getTableRows).toHaveBeenCalledTimes(2));
        fireEvent.click(screen.getByRole('button', {name: 'Next'}));
        await screen.findByText('customers page 2');
        fireEvent.change(screen.getByPlaceholderText(/Search displayed columns/), {target: {value: 'zz'}});

        api.getTableRows.mockClear();
        select('orders');
        await screen.findByText('orders page 1');
        expect(api.getTableRows).toHaveBeenCalledTimes(1);
        expect(api.getTableRows).toHaveBeenCalledWith(
            expect.objectContaining({table: 'orders', page: 1, sortColumn: '', sortDesc: false, filter: ''}),
        );
        expect(screen.getByPlaceholderText(/Search displayed columns/)).toHaveValue('');
    });

    it('resets to page 1 with a single fetch when sorting', async () => {
        renderGrid('customers');
        await screen.findByText('customers page 1');
        fireEvent.click(screen.getByRole('button', {name: 'Next'}));
        await screen.findByText('customers page 2');

        api.getTableRows.mockClear();
        fireEvent.click(screen.getByRole('button', {name: 'id'}));
        await screen.findByText('customers page 1');
        expect(api.getTableRows).toHaveBeenCalledTimes(1);
        expect(api.getTableRows).toHaveBeenCalledWith(expect.objectContaining({page: 1, sortColumn: 'id'}));
    });

    it('ignores a slow response for a table that is no longer selected', async () => {
        let resolveSlow: (v: model.TableRowsResponse) => void = () => {};
        api.getTableRows.mockImplementation((req: model.TableRowsRequest) =>
            req.table === 'customers'
                ? new Promise((resolve) => {
                      resolveSlow = resolve;
                  })
                : Promise.resolve(rowsFor(req)),
        );
        const {select} = renderGrid('customers');
        await waitFor(() => expect(api.getTableRows).toHaveBeenCalledTimes(1));

        select('orders');
        await screen.findByText('orders page 1');
        await act(async () => {
            resolveSlow(rowsFor({table: 'customers', page: 1, pageSize: 100} as model.TableRowsRequest, 'stale'));
        });
        expect(screen.queryByText('stale page 1')).toBeNull();
        expect(screen.getByText('orders page 1')).toBeInTheDocument();
    });

    it('exports every row and reports where it went', async () => {
        api.exportTable.mockResolvedValue({path: '/tmp/customers.csv', rowCount: 500});
        renderGrid('customers');
        await screen.findByText('customers page 1');

        fireEvent.click(screen.getByRole('button', {name: 'Export all rows'}));
        expect(await screen.findByText('Exported 500 rows to /tmp/customers.csv')).toBeInTheDocument();
        expect(api.exportTable).toHaveBeenCalledWith(expect.objectContaining({table: 'customers'}), 'all');
    });

    it('shows SQLite error detail', async () => {
        api.getTableRows.mockRejectedValue(new Error(JSON.stringify({
            code: 'MALFORMED_SQL', message: 'The query could not be executed.', detail: 'database disk image is malformed',
        })));
        renderGrid('customers');
        expect(await screen.findByText('The query could not be executed.')).toBeInTheDocument();
        expect(screen.getByText('database disk image is malformed')).toBeInTheDocument();
    });
});
