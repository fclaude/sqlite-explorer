import {useCallback, useEffect, useRef, useState} from 'react';
import {formatAPIError, model, WailsAPI} from '../api';
import {useDebouncedLoading} from '../hooks/useDebouncedLoading';
import {useApp} from '../state/AppProvider';
import {addQueryHistory, getQueryHistory} from '../state/history';
import {CellDetailContext, RecordRowContext} from '../utils/cellDetail';
import {CellDetailModal} from './CellDetailModal';
import {RecordDetailModal} from './RecordDetailModal';
import {GridCell} from './GridCell';
import {LoadingOverlay} from './LoadingOverlay';
import './DetailModal.css';
import './SqlEditor.css';

interface SqlEditorProps {
    hasDatabase: boolean;
    onOpenDatabase: () => void;
}

export function SqlEditor({hasDatabase, onOpenDatabase}: SqlEditorProps) {
    const {reportTableQuery, clearError} = useApp();
    const [sql, setSql] = useState('SELECT * FROM customers LIMIT 5;');
    const [result, setResult] = useState<model.QueryResponse | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);
    const [exporting, setExporting] = useState(false);
    const [history, setHistory] = useState<string[]>([]);
    const runRef = useRef(0);
    const showLoading = useDebouncedLoading(loading);
    const [cellDetail, setCellDetail] = useState<CellDetailContext | null>(null);
    const [recordDetail, setRecordDetail] = useState<RecordRowContext | null>(null);

    const refreshHistory = useCallback(() => {
        setHistory(getQueryHistory());
    }, []);

    useEffect(() => {
        refreshHistory();
    }, [refreshHistory]);

    const runQuery = useCallback(async () => {
        const text = sql.trim();
        if (!text) {
            return;
        }
        const runId = ++runRef.current;
        setLoading(true);
        setError(null);
        clearError();
        try {
            const resp = await WailsAPI.runQuery({sql: text});
            if (runId !== runRef.current) {
                return;
            }
            setResult(resp);
            addQueryHistory(text);
            refreshHistory();
            const trunc = resp.truncated ? ' (truncated)' : '';
            reportTableQuery(
                resp.durationMs ?? 0,
                `${resp.rowCount ?? 0} rows${trunc} | ${resp.durationMs ?? 0}ms`,
                !!resp.truncated,
            );
        } catch (err) {
            if (runId !== runRef.current) {
                return;
            }
            const {message} = formatAPIError(err);
            setError(message);
            setResult(null);
            reportTableQuery(0, null, false);
        } finally {
            if (runId === runRef.current) {
                setLoading(false);
            }
        }
    }, [sql, clearError, reportTableQuery, refreshHistory]);

    const cancelQuery = useCallback(async () => {
        try {
            await WailsAPI.cancelQuery(0);
        } catch {
            // ignore cancel errors
        }
    }, []);

    const exportCSV = async () => {
        const text = sql.trim();
        if (!text) {
            return;
        }
        setExporting(true);
        try {
            await WailsAPI.exportQueryResult(text);
        } catch (err) {
            setError(formatAPIError(err).message);
        } finally {
            setExporting(false);
        }
    };

    const onKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
        if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
            e.preventDefault();
            runQuery();
        }
    };

    const buildRecordContext = (rowIndex: number, cells: model.CellValue[]): RecordRowContext | null => {
        if (!result?.columns) {
            return null;
        }
        return {
            source: 'query',
            rowIndex,
            columns: result.columns,
            cells,
        };
    };

    const openCellDetail = (rowIndex: number, columnIndex: number) => {
        if (!result?.columns || !result.rows?.[rowIndex]) {
            return;
        }
        const col = result.columns[columnIndex];
        setCellDetail({
            source: 'query',
            rowIndex,
            columns: result.columns,
            cells: result.rows[rowIndex],
            columnIndex,
            columnName: col.name,
            cell: result.rows[rowIndex][columnIndex],
        });
    };

    const openRecordDetail = (rowIndex: number) => {
        const row = result?.rows?.[rowIndex];
        if (!row) {
            return;
        }
        const ctx = buildRecordContext(rowIndex, row);
        if (ctx) {
            setRecordDetail(ctx);
        }
    };

    if (!hasDatabase) {
        return (
            <div className="panel-empty">
                <h2 className="panel-empty-title">No database open</h2>
                <p>Open a SQLite file to run SQL queries.</p>
                <button type="button" className="btn btn-primary" onClick={onOpenDatabase}>
                    Open database
                </button>
            </div>
        );
    }

    return (
        <div className="sql-editor panel-with-overlay">
            <div className="sql-editor-toolbar">
                <button type="button" className="btn btn-primary" onClick={runQuery} disabled={loading}>
                    Run query
                </button>
                <button
                    type="button"
                    className="btn"
                    onClick={cancelQuery}
                    disabled={!loading}
                >
                    Cancel
                </button>
                <button
                    type="button"
                    className="btn"
                    onClick={exportCSV}
                    disabled={loading || exporting || !sql.trim()}
                >
                    Export CSV
                </button>
                <span className="sql-editor-hint">Cmd/Ctrl+Enter to run</span>
                {history.length > 0 && (
                    <label className="sql-history-label">
                        History
                        <select
                            className="sql-history-select"
                            defaultValue=""
                            onChange={(e) => {
                                if (e.target.value) {
                                    setSql(e.target.value);
                                }
                                e.target.value = '';
                            }}
                        >
                            <option value="">Recent queries...</option>
                            {history.slice(0, 10).map((q) => (
                                <option key={q} value={q}>
                                    {q.length > 60 ? `${q.slice(0, 60)}...` : q}
                                </option>
                            ))}
                        </select>
                    </label>
                )}
            </div>
            <textarea
                className="sql-textarea"
                value={sql}
                onChange={(e) => setSql(e.target.value)}
                onKeyDown={onKeyDown}
                placeholder="SELECT * FROM ..."
                spellCheck={false}
            />
            {error && <p className="sql-error" role="alert">{error}</p>}
            {result?.truncated && (
                <p className="sql-truncated">Showing first {result.rowCount} rows (limit 1000).</p>
            )}
            <div className="sql-results-area">
                {showLoading && <LoadingOverlay label="Running query..." />}
                {result && result.columns && result.columns.length > 0 && (
                    <div className="sql-results-wrap">
                        <table className="sql-results-table">
                            <thead>
                                <tr>
                                    <th className="data-grid-row-num" scope="col" aria-label="Row" />
                                    {result.columns.map((col) => (
                                        <th key={col.name}>{col.name}</th>
                                    ))}
                                </tr>
                            </thead>
                            <tbody>
                                {result.rows?.length === 0 && (
                                    <tr>
                                        <td colSpan={result.columns.length + 1} className="sql-empty-row">
                                            No rows returned.
                                        </td>
                                    </tr>
                                )}
                                {result.rows?.map((row, ri) => (
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
                )}
            </div>

            <CellDetailModal
                context={cellDetail}
                onClose={() => setCellDetail(null)}
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
