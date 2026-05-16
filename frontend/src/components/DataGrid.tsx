import {useCallback, useEffect, useState} from 'react';
import {blobCellTooltip, formatAPIError, formatCellDisplay, model, WailsAPI} from '../api';
import {useApp} from '../state/AppProvider';
import {SelectedObject} from '../state/types';
import './DataGrid.css';

const PAGE_SIZES = [50, 100, 500, 1000];

interface DataGridProps {
    selected: SelectedObject | null;
    hasDatabase: boolean;
    onOpenDatabase: () => void;
}

export function DataGrid({selected, hasDatabase, onOpenDatabase}: DataGridProps) {
    const {reportTableQuery} = useApp();
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(100);
    const [sortColumn, setSortColumn] = useState('');
    const [sortDesc, setSortDesc] = useState(false);
    const [filter, setFilter] = useState('');
    const [filterInput, setFilterInput] = useState('');
    const [data, setData] = useState<model.TableRowsResponse | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const tableName =
        selected && (selected.kind === 'table' || selected.kind === 'view') ? selected.name : null;

    const loadRows = useCallback(async () => {
        if (!tableName) {
            setData(null);
            return;
        }
        setLoading(true);
        setError(null);
        try {
            const resp = await WailsAPI.getTableRows({
                table: tableName,
                page,
                pageSize,
                sortColumn,
                sortDesc,
                filter,
                withTotal: true,
            });
            setData(resp);
            const total = resp.totalRows ?? undefined;
            const pageInfo = total != null
                ? `Page ${resp.page} | rows ${resp.rows?.length ?? 0} | total ${total}`
                : `Page ${resp.page} | rows ${resp.rows?.length ?? 0}`;
            reportTableQuery(resp.durationMs ?? 0, pageInfo);
        } catch (err) {
            const {message} = formatAPIError(err);
            setError(message);
            setData(null);
            reportTableQuery(0, null);
        } finally {
            setLoading(false);
        }
    }, [tableName, page, pageSize, sortColumn, sortDesc, filter, reportTableQuery]);

    useEffect(() => {
        setPage(1);
        setSortColumn('');
        setSortDesc(false);
        setFilter('');
        setFilterInput('');
    }, [tableName]);

    useEffect(() => {
        const t = setTimeout(() => setFilter(filterInput), 300);
        return () => clearTimeout(t);
    }, [filterInput]);

    useEffect(() => {
        setPage(1);
    }, [pageSize, filter, sortColumn, sortDesc]);

    useEffect(() => {
        loadRows();
    }, [loadRows]);

    if (!hasDatabase) {
        return (
            <div className="panel-empty">
                <h2 className="panel-empty-title">No database open</h2>
                <p>Open a SQLite file to browse table data.</p>
                <button type="button" className="btn btn-primary" onClick={onOpenDatabase}>
                    Open database
                </button>
            </div>
        );
    }

    if (!tableName) {
        return (
            <div className="data-grid">
                <p className="panel-hint">Select a table or view in the sidebar to browse rows.</p>
            </div>
        );
    }

    const total = data?.totalRows;
    const maxPage = total != null ? Math.max(1, Math.ceil(total / pageSize)) : null;

    const toggleSort = (col: string) => {
        if (sortColumn === col) {
            setSortDesc(!sortDesc);
        } else {
            setSortColumn(col);
            setSortDesc(false);
        }
    };

    return (
        <div className="data-grid">
            <div className="data-grid-toolbar">
                <span className="data-grid-title">{tableName}</span>
                <label className="data-grid-control">
                    Filter
                    <input
                        type="text"
                        value={filterInput}
                        onChange={(e) => setFilterInput(e.target.value)}
                        placeholder="Search displayed columns..."
                        className="data-grid-input"
                    />
                </label>
                <label className="data-grid-control">
                    Page size
                    <select
                        value={pageSize}
                        onChange={(e) => setPageSize(Number(e.target.value))}
                        className="data-grid-select"
                    >
                        {PAGE_SIZES.map((n) => (
                            <option key={n} value={n}>{n}</option>
                        ))}
                    </select>
                </label>
                <div className="data-grid-pager">
                    <button
                        type="button"
                        className="btn"
                        disabled={page <= 1 || loading}
                        onClick={() => setPage((p) => p - 1)}
                    >
                        Previous
                    </button>
                    <span className="data-grid-page-label">
                        Page {page}
                        {maxPage != null ? ` of ${maxPage}` : ''}
                    </span>
                    <button
                        type="button"
                        className="btn"
                        disabled={loading || (maxPage != null && page >= maxPage)}
                        onClick={() => setPage((p) => p + 1)}
                    >
                        Next
                    </button>
                </div>
            </div>

            {error && <p className="data-grid-error">{error}</p>}
            {loading && <p className="data-grid-loading">Loading...</p>}

            <div className="data-grid-table-wrap">
                <table className="data-grid-table">
                    <thead>
                        <tr>
                            {data?.columns?.map((col) => (
                                <th key={col.name}>
                                    <button
                                        type="button"
                                        className="data-grid-sort-btn"
                                        onClick={() => toggleSort(col.name)}
                                    >
                                        {col.name}
                                        {sortColumn === col.name ? (sortDesc ? ' v' : ' ^') : ''}
                                    </button>
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {!loading && data?.rows?.length === 0 && (
                            <tr>
                                <td colSpan={data.columns?.length ?? 1} className="data-grid-empty-row">
                                    No rows.
                                </td>
                            </tr>
                        )}
                        {data?.rows?.map((row, ri) => (
                            <tr key={ri}>
                                {row.map((cell, ci) => (
                                    <td
                                        key={ci}
                                        title={blobCellTooltip(cell)}
                                        className={cell.kind === 'blob' ? 'cell-blob' : undefined}
                                    >
                                        {formatCellDisplay(cell)}
                                    </td>
                                ))}
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
}
