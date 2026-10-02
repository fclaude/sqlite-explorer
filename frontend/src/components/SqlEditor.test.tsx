import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import {beforeEach, describe, expect, it, vi} from 'vitest';
import {model} from '../../wailsjs/go/models';
import {AppProvider} from '../state/AppProvider';
import {SqlEditor} from './SqlEditor';

const api = vi.hoisted(() => ({
    runQuery: vi.fn(),
    cancelQuery: vi.fn(),
    exportQueryResult: vi.fn(),
    getStatementCategories: vi.fn(),
    getSchema: vi.fn(),
}));

vi.mock('../api', async (importOriginal) => {
    const actual = await importOriginal<typeof import('../api')>();
    return {...actual, WailsAPI: api};
});

const categories: model.StatementCategory[] = [
    {id: 'read', label: 'Read queries', description: 'Always allowed.', statements: ['SELECT'], pragmas: ['table_info'], configurable: false},
    {id: 'data', label: 'Data changes', description: 'Rows.', statements: ['INSERT', 'UPDATE', 'DELETE'], configurable: true},
    {id: 'schema', label: 'Schema changes', description: 'Tables.', statements: ['CREATE', 'DROP'], configurable: true},
    {id: 'forbidden', label: 'Never allowed', description: 'Corrupts.', statements: ['PRAGMA writable_schema'], configurable: false},
];

const STORAGE_KEY = 'sqlite-explorer.sqlPermissions';

function renderEditor() {
    return render(
        <AppProvider>
            <SqlEditor hasDatabase={true} onOpenDatabase={() => {}} />
        </AppProvider>,
    );
}

function setSQL(text: string) {
    fireEvent.change(screen.getByLabelText('SQL'), {target: {value: text}});
}

describe('SqlEditor permissions', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.clear();
        api.getStatementCategories.mockResolvedValue(categories);
        api.getSchema.mockResolvedValue({tables: [], views: [], indexes: [], triggers: []});
        api.runQuery.mockResolvedValue({
            columns: [{name: 'n', type: ''}], rows: [[{kind: 'int', value: 1}]], rowCount: 1,
            truncated: false, durationMs: 1, queryId: 1, statementCount: 1, rowsAffected: 0,
            changed: false, schemaChanged: false,
        });
    });

    it('runs read-only by default and lists what is allowed', async () => {
        renderEditor();
        fireEvent.click(await screen.findByRole('button', {name: /Read-only · Permissions/}));
        expect(await screen.findByText('Data changes')).toBeInTheDocument();
        expect(screen.getByText(/PRAGMA writable_schema/)).toBeInTheDocument();
        expect(screen.getByText('table_info')).toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', {name: 'Run query'}));
        await waitFor(() => expect(api.runQuery).toHaveBeenCalledWith({sql: 'SELECT * FROM customers LIMIT 5;', allow: []}));
    });

    it('sends and remembers the categories the user allows', async () => {
        renderEditor();
        fireEvent.click(await screen.findByRole('button', {name: /Permissions/}));
        fireEvent.click(await screen.findByRole('checkbox', {name: /Data changes/}));
        expect(JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '[]')).toEqual(['data']);
        expect(screen.getByRole('button', {name: /Allowed: Data changes/})).toBeInTheDocument();

        setSQL('DELETE FROM t');
        fireEvent.click(screen.getByRole('button', {name: 'Run query'}));
        await waitFor(() => expect(api.runQuery).toHaveBeenCalledWith({sql: 'DELETE FROM t', allow: ['data']}));
    });

    it('ignores stored permissions the backend does not offer', async () => {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(['data', 'everything']));
        renderEditor();
        await screen.findByRole('button', {name: /Allowed: Data changes/});
        fireEvent.click(screen.getByRole('button', {name: 'Run query'}));
        await waitFor(() => expect(api.runQuery).toHaveBeenCalledWith(expect.objectContaining({allow: ['data']})));
    });

    it('offers to allow a blocked category', async () => {
        api.runQuery.mockRejectedValueOnce(new Error(JSON.stringify({
            code: 'READ_ONLY_VIOLATION',
            message: 'DELETE is not allowed. Enable "Data changes" under Permissions to run it.',
            detail: 'data',
        })));
        renderEditor();
        await screen.findByRole('button', {name: /Read-only/});
        setSQL('DELETE FROM t');
        fireEvent.click(screen.getByRole('button', {name: 'Run query'}));

        const allow = await screen.findByRole('button', {name: 'Allow Data changes'});
        expect(screen.queryByText('data', {exact: true})).toBeNull();
        fireEvent.click(allow);
        expect(screen.getByRole('checkbox', {name: /Data changes/})).toBeChecked();
        expect(screen.queryByRole('alert')).toBeNull();
    });

    it('reports changed rows and refreshes the schema after schema changes', async () => {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(['data', 'schema']));
        api.runQuery.mockResolvedValueOnce({
            columns: [], rows: [], rowCount: 0, truncated: false, durationMs: 1, queryId: 2,
            statementCount: 2, rowsAffected: 3, changed: true, schemaChanged: true,
        });
        renderEditor();
        await screen.findByRole('button', {name: /Allowed: Data changes, Schema changes/});
        setSQL('CREATE TABLE u(a); INSERT INTO u VALUES (1),(2),(3)');
        fireEvent.click(screen.getByRole('button', {name: 'Run query'}));

        expect(await screen.findByText('2 statements ran, 3 rows changed.')).toBeInTheDocument();
        await waitFor(() => expect(api.getSchema).toHaveBeenCalled());
    });

    it('shows the SQLite error detail', async () => {
        api.runQuery.mockRejectedValueOnce(new Error(JSON.stringify({
            code: 'MALFORMED_SQL',
            message: 'The SQL could not be parsed. Check your syntax.',
            detail: 'SQL logic error: near "FORM": syntax error (1)',
        })));
        renderEditor();
        await screen.findByRole('button', {name: /Read-only/});
        fireEvent.click(screen.getByRole('button', {name: 'Run query'}));
        expect(await screen.findByText('SQL logic error: near "FORM": syntax error (1)')).toBeInTheDocument();
    });

    it('reports exports and lets them be cancelled', async () => {
        let finish: (v: model.ExportResult) => void = () => {};
        api.exportQueryResult.mockReturnValueOnce(new Promise((resolve) => {
            finish = resolve;
        }));
        renderEditor();
        await screen.findByRole('button', {name: /Read-only/});
        fireEvent.click(screen.getByRole('button', {name: 'Export CSV'}));

        const cancel = await screen.findByRole('button', {name: 'Cancel'});
        expect(cancel).not.toBeDisabled();
        fireEvent.click(cancel);
        expect(api.cancelQuery).toHaveBeenCalledWith(0);

        finish({path: '/tmp/out.csv', rowCount: 1});
        expect(await screen.findByText('Exported 1 row to /tmp/out.csv')).toBeInTheDocument();
    });
});
