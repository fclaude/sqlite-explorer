import {useCallback, useEffect, useState} from 'react';
import {blobCellTooltip, formatAPIError, formatCellDisplay, model, WailsAPI} from '../api';
import {useApp} from '../state/AppProvider';
import {addQueryHistory, getQueryHistory} from '../state/history';
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
        setLoading(true);
        setError(null);
        clearError();
        try {
            const resp = await WailsAPI.runQuery({sql: text});
            setResult(resp);
            addQueryHistory(text);
            refreshHistory();
            const trunc = resp.truncated ? ' (truncated)' : '';
            reportTableQuery(
                resp.durationMs ?? 0,
                `${resp.rowCount ?? 0} rows${trunc} | ${resp.durationMs ?? 0}ms`,
            );
        } catch (err) {
            const {message} = formatAPIError(err);
            setError(message);
            setResult(null);
            reportTableQuery(0, null);
        } finally {
            setLoading(false);
        }
    }, [sql, clearError, reportTableQuery, refreshHistory]);

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
        <div className="sql-editor">
            <div className="sql-editor-toolbar">
                <button type="button" className="btn btn-primary" onClick={runQuery} disabled={loading}>
                    Run query
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
            {loading && <p className="sql-status">Running...</p>}
            {error && <p className="sql-error">{error}</p>}
            {result?.truncated && (
                <p className="sql-truncated">Showing first {result.rowCount} rows (limit 1000).</p>
            )}
            {result && result.columns && result.columns.length > 0 && (
                <div className="sql-results-wrap">
                    <table className="sql-results-table">
                        <thead>
                            <tr>
                                {result.columns.map((col) => (
                                    <th key={col.name}>{col.name}</th>
                                ))}
                            </tr>
                        </thead>
                        <tbody>
                            {result.rows?.length === 0 && (
                                <tr>
                                    <td colSpan={result.columns.length} className="sql-empty-row">
                                        No rows returned.
                                    </td>
                                </tr>
                            )}
                            {result.rows?.map((row, ri) => (
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
            )}
        </div>
    );
}
