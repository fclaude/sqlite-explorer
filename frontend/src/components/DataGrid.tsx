import {useCallback, useEffect, useState} from 'react';
import {formatAPIError, model, WailsAPI} from '../api';
import {useDebouncedLoading} from '../hooks/useDebouncedLoading';
import {useApp} from '../state/AppProvider';
import {buildColumnUpdates, draftToColumnUpdate} from '../utils/recordSave';
import {LoadingOverlay} from './LoadingOverlay';
import {CellDetailModal} from './CellDetailModal';
import {RecordDetailModal} from './RecordDetailModal';
import {GridCell} from './GridCell';
import {CellDetailContext, RecordRowContext} from '../utils/cellDetail';
import {SelectedObject} from '../state/types';
import './DataGrid.css';
import './DetailModal.css';

const PAGE_SIZES = [50, 100, 500, 1000];

interface DataGridProps {
    selected: SelectedObject | null;
    hasDatabase: boolean;
    onOpenDatabase: () => void;
}

export function DataGrid({selected, hasDatabase, onOpenDatabase}: DataGridProps) {
    const {reportTableQuery, state} = useApp();
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(100);
    const [sortColumn, setSortColumn] = useState('');
    const [sortDesc, setSortDesc] = useState(false);
    const [filter, setFilter] = useState('');
    const [filterInput, setFilterInput] = useState('');
    const [data, setData] = useState<model.TableRowsResponse | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [exporting, setExporting] = useState(false);
    const showLoading = useDebouncedLoading(loading);
    const [cellDetail, setCellDetail] = useState<CellDetailContext | null>(null);
    const [recordDetail, setRecordDetail] = useState<RecordRowContext | null>(null);

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
            reportTableQuery(resp.durationMs ?? 0, pageInfo, false);
        } catch (err) {
            const {message} = formatAPIError(err);
            setError(message);
            setData(null);
            reportTableQuery(0, null, false);
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

    const exportCSV = async () => {
        if (!tableName) {
            return;
        }
        setExporting(true);
        try {
            await WailsAPI.exportTablePage({
                table: tableName,
                page,
                pageSize,
                sortColumn,
                sortDesc,
                filter,
                withTotal: false,
            });
        } catch (err) {
            setError(formatAPIError(err).message);
        } finally {
            setExporting(false);
        }
    };

    const toggleSort = (col: string) => {
        if (sortColumn === col) {
            setSortDesc(!sortDesc);
        } else {
            setSortColumn(col);
            setSortDesc(false);
        }
    };

    const columnMeta =
        state.schema?.tables?.find((t) => t.name === tableName)?.columns ?? [];

    const buildRecordContext = (rowIndex: number, cells: model.CellValue[]): RecordRowContext | null => {
        if (!data?.columns) {
            return null;
        }
        return {
            source: 'table',
            tableName: tableName ?? undefined,
            rowIndex,
            page,
            rowId: data.rowIds?.[rowIndex],
            editable: data.editable && data.rowIds?.[rowIndex] != null,
            columnMeta,
            columns: data.columns,
            cells,
        };
    };

    const saveRecord = async (ctx: RecordRowContext, drafts: Record<string, string>) => {
        if (!tableName || ctx.rowId == null) {
            throw new Error('This row cannot be saved.');
        }
        const updates = buildColumnUpdates(ctx.columns, ctx.cells, drafts, ctx.columnMeta ?? columnMeta);
        if (updates.length === 0) {
            return;
        }
        await WailsAPI.updateTableRow({
            table: tableName,
            rowId: ctx.rowId,
            updates,
        });
        await loadRows();
    };

    const saveCellField = async (ctx: CellDetailContext, draft: string) => {
        if (!tableName || ctx.rowId == null) {
            throw new Error('This field cannot be saved.');
        }
        const meta = (ctx.columnMeta ?? columnMeta).find((c) => c.name === ctx.columnName);
        if (!meta) {
            throw new Error(`Unknown column ${ctx.columnName}.`);
        }
        const update = draftToColumnUpdate(ctx.columnName, draft, ctx.cell, meta);
        await WailsAPI.updateTableRow({
            table: tableName,
            rowId: ctx.rowId,
            updates: [update],
        });
        await loadRows();
    };

    const openCellDetail = (rowIndex: number, columnIndex: number) => {
        const row = data?.rows?.[rowIndex];
        if (!row || !data?.columns) {
            return;
        }
        const base = buildRecordContext(rowIndex, row);
        if (!base) {
            return;
        }
        setCellDetail({
            ...base,
            columnIndex,
            columnName: data.columns[columnIndex].name,
            cell: row[columnIndex],
        });
    };

    const openRecordDetail = (rowIndex: number) => {
        const row = data?.rows?.[rowIndex];
        if (!row) {
            return;
        }
        const ctx = buildRecordContext(rowIndex, row);
        if (ctx) {
            setRecordDetail(ctx);
        }
    };

    return (
        <div className="data-grid panel-with-overlay">
            <div className="data-grid-toolbar">
                <span className="data-grid-title">
                    {tableName}
                    <span className="data-grid-hint"> · click cell · double-click row</span>
                </span>
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
                <button
                    type="button"
                    className="btn"
                    disabled={loading || exporting || !data?.rows?.length}
                    onClick={exportCSV}
                >
                    Export CSV
                </button>
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

            {error && <p className="data-grid-error" role="alert">{error}</p>}

            <div className="data-grid-table-wrap">
                {showLoading && <LoadingOverlay label="Loading rows..." />}
                <table className="data-grid-table">
                    <thead>
                        <tr>
                            <th className="data-grid-row-num" scope="col" aria-label="Row" />
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
                                <td colSpan={(data.columns?.length ?? 0) + 1} className="data-grid-empty-row">
                                    No rows.
                                </td>
                            </tr>
                        )}
                        {data?.rows?.map((row, ri) => (
                            <tr
                                key={ri}
                                onDoubleClick={() => openRecordDetail(ri)}
                                title="Double-click to view full row"
                            >
                                <td className="data-grid-row-num">
                                    <button
                                        type="button"
                                        className="data-grid-row-btn"
                                        onClick={() => openRecordDetail(ri)}
                                        title="View full row"
                                    >
                                        {ri + 1}
                                    </button>
                                </td>
                                {row.map((cell, ci) => (
                                    <td key={ci}>
                                        <GridCell cell={cell} onOpen={() => openCellDetail(ri, ci)} />
                                    </td>
                                ))}
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>

            <CellDetailModal
                context={cellDetail}
                onClose={() => setCellDetail(null)}
                onSave={
                    cellDetail?.editable
                        ? (draft) => saveCellField(cellDetail, draft)
                        : undefined
                }
                onViewRow={
                    cellDetail
                        ? () => {
                              const ctx = buildRecordContext(cellDetail.rowIndex, cellDetail.cells);
                              setCellDetail(null);
                              if (ctx) {
                                  setRecordDetail(ctx);
                              }
                          }
                        : undefined
                }
            />
            <RecordDetailModal
                context={recordDetail}
                onClose={() => setRecordDetail(null)}
                onSave={
                    recordDetail?.editable
                        ? (drafts) => saveRecord(recordDetail, drafts)
                        : undefined
                }
                onOpenCell={(columnIndex) => {
                    if (!recordDetail) {
                        return;
                    }
                    setRecordDetail(null);
                    setCellDetail({
                        ...recordDetail,
                        columnIndex,
                        columnName: recordDetail.columns[columnIndex].name,
                        cell: recordDetail.cells[columnIndex],
                    });
                }}
            />
        </div>
    );
}
