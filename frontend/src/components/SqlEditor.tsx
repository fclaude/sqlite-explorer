import {useCallback, useEffect, useRef, useState} from 'react';
import {APIError, formatAPIError, model, WailsAPI} from '../api';
import {useDebouncedLoading} from '../hooks/useDebouncedLoading';
import {useApp} from '../state/AppProvider';
import {addQueryHistory, getQueryHistory} from '../state/history';
import {loadAllowedCategories, saveAllowedCategories} from '../state/permissions';
import {CellDetailContext, RecordRowContext} from '../utils/cellDetail';
import {CellDetailModal} from './CellDetailModal';
import {ErrorNotice} from './ErrorNotice';
import {RecordDetailModal} from './RecordDetailModal';
import {GridCell} from './GridCell';
import {LoadingOverlay} from './LoadingOverlay';
import {StatementPermissions} from './StatementPermissions';
import './DetailModal.css';
import './SqlEditor.css';

interface SqlEditorProps {
    hasDatabase: boolean;
    onOpenDatabase: () => void;
}

function plural(n: number, word: string): string {
    return `${n} ${word}${n === 1 ? '' : 's'}`;
}

export function SqlEditor({hasDatabase, onOpenDatabase}: SqlEditorProps) {
    const {reportTableQuery, clearError, refreshSchema, state} = useApp();
    const [sql, setSql] = useState('SELECT * FROM customers LIMIT 5;');
    const [result, setResult] = useState<model.QueryResponse | null>(null);
    const [error, setError] = useState<APIError | null>(null);
    const [notice, setNotice] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);
    const [exporting, setExporting] = useState(false);
    const [history, setHistory] = useState<string[]>([]);
    const [categories, setCategories] = useState<model.StatementCategory[]>([]);
    const [allowed, setAllowed] = useState<string[]>(loadAllowedCategories);
    const [showPermissions, setShowPermissions] = useState(false);
    const runRef = useRef(0);
    const showLoading = useDebouncedLoading(loading);
    const [cellDetail, setCellDetail] = useState<CellDetailContext | null>(null);
    const [recordDetail, setRecordDetail] = useState<RecordRowContext | null>(null);

    const readOnlyDatabase = !!state.dbInfo?.readOnly;
    const configurable = categories.filter((c) => c.configurable);
    // Ignore stored ids this version of the backend does not know.
    const activeAllowed = readOnlyDatabase ? [] : allowed.filter((id) => configurable.some((c) => c.id === id));
    const activeLabels = configurable.filter((c) => activeAllowed.includes(c.id)).map((c) => c.label);

    const refreshHistory = useCallback(() => {
        setHistory(getQueryHistory());
    }, []);

    useEffect(() => {
        refreshHistory();
    }, [refreshHistory]);

    useEffect(() => {
        let cancelled = false;
        WailsAPI.getStatementCategories()
            .then((cats) => {
                if (!cancelled) {
                    setCategories(cats);
                }
            })
            .catch(() => {
                // Without the list, only read queries run; the backend still enforces this.
            });
        return () => {
            cancelled = true;
        };
    }, []);

    const setCategoryAllowed = useCallback((id: string, on: boolean) => {
        setAllowed((prev) => {
            const next = on ? [...new Set([...prev, id])] : prev.filter((x) => x !== id);
            saveAllowedCategories(next);
            return next;
        });
    }, []);

    const runQuery = useCallback(async () => {
        const text = sql.trim();
        if (!text) {
            return;
        }
        const runId = ++runRef.current;
        const allow = activeAllowed;
        setLoading(true);
        setError(null);
        setNotice(null);
        clearError();
        try {
            const resp = await WailsAPI.runQuery({sql: text, allow});
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
            if (resp.schemaChanged) {
                void refreshSchema();
            }
        } catch (err) {
            if (runId !== runRef.current) {
                return;
            }
            setError(formatAPIError(err));
            setResult(null);
            reportTableQuery(0, null, false);
            // A failed script may still have changed the schema before the failing statement.
            if (allow.includes('schema')) {
                void refreshSchema();
            }
        } finally {
            if (runId === runRef.current) {
                setLoading(false);
            }
        }
    }, [sql, activeAllowed, clearError, reportTableQuery, refreshHistory, refreshSchema]);

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
        setError(null);
        setNotice(null);
        try {
            const res = await WailsAPI.exportQueryResult(text);
            if (res.path) {
                setNotice(`Exported ${plural(res.rowCount, 'row')} to ${res.path}`);
            }
        } catch (err) {
            const apiErr = formatAPIError(err);
            if (apiErr.code === 'CANCELLED') {
                setNotice('Export cancelled.');
            } else {
                setError(apiErr);
            }
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

    // A blocked statement names its category in the error detail; offer to allow it.
    const blockedCategory =
        error?.code === 'READ_ONLY_VIOLATION' && !readOnlyDatabase
            ? configurable.find((c) => c.id === error.detail && !activeAllowed.includes(c.id))
            : undefined;

    const resultSummary = (() => {
        if (!result) {
            return null;
        }
        const parts: string[] = [];
        if (result.changed) {
            parts.push(`${plural(result.statementCount, 'statement')} ran`);
            parts.push(`${plural(result.rowsAffected, 'row')} changed`);
        } else if (!result.columns?.length) {
            parts.push(`${plural(result.statementCount, 'statement')} ran`);
        }
        return parts.length ? `${parts.join(', ')}.` : null;
    })();

    return (
        <div className="sql-editor panel-with-overlay">
            <div className="sql-editor-toolbar">
                <button type="button" className="btn btn-primary" onClick={runQuery} disabled={loading || exporting}>
                    Run query
                </button>
                <button
                    type="button"
                    className="btn"
                    onClick={cancelQuery}
                    disabled={!loading && !exporting}
                >
                    Cancel
                </button>
                <button
                    type="button"
                    className="btn"
                    onClick={exportCSV}
                    disabled={loading || exporting || !sql.trim()}
                    title="Run this read query again and export every row to CSV"
                >
                    {exporting ? 'Exporting…' : 'Export CSV'}
                </button>
                <button
                    type="button"
                    className={`btn sql-permissions-btn${activeAllowed.length ? ' sql-permissions-btn-writes' : ''}`}
                    onClick={() => setShowPermissions((v) => !v)}
                    aria-expanded={showPermissions}
                    aria-controls="sql-permissions"
                >
                    {activeAllowed.length ? `Allowed: ${activeLabels.join(', ')}` : 'Read-only'}
                    {' · Permissions'}
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
            {showPermissions && (
                <StatementPermissions
                    id="sql-permissions"
                    categories={categories}
                    allowed={activeAllowed}
                    readOnlyDatabase={readOnlyDatabase}
                    onChange={setCategoryAllowed}
                />
            )}
            <textarea
                className={`sql-textarea${activeAllowed.length ? ' sql-textarea-writes' : ''}`}
                value={sql}
                onChange={(e) => setSql(e.target.value)}
                onKeyDown={onKeyDown}
                placeholder="SELECT * FROM ..."
                spellCheck={false}
                aria-label="SQL"
            />
            <ErrorNotice error={error} className="sql-error">
                {blockedCategory && (
                    <div>
                        <button
                            type="button"
                            className="btn"
                            onClick={() => {
                                setCategoryAllowed(blockedCategory.id, true);
                                setShowPermissions(true);
                                setError(null);
                            }}
                        >
                            Allow {blockedCategory.label}
                        </button>
                    </div>
                )}
            </ErrorNotice>
            {notice && <p className="sql-status" role="status">{notice}</p>}
            {resultSummary && <p className="sql-status" role="status">{resultSummary}</p>}
            {result?.truncated && (
                <p className="sql-truncated">
                    Showing the first {result.rowCount} rows. Export CSV includes every row.
                </p>
            )}
            <div className="sql-results-area">
                {showLoading && <LoadingOverlay label="Running query..." />}
                {result && result.columns && result.columns.length > 0 && (
                    <div className="sql-results-wrap">
                        <table className="sql-results-table">
                            <thead>
                                <tr>
                                    <th className="data-grid-row-num" scope="col" aria-label="Row" />
                                    {result.columns.map((col, ci) => (
                                        <th key={`${ci}-${col.name}`}>{col.name}</th>
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
