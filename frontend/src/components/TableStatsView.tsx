import {useCallback, useEffect, useRef, useState} from 'react';
import {APIError, formatAPIError, model, WailsAPI} from '../api';
import {useDebouncedLoading} from '../hooks/useDebouncedLoading';
import {formatBytes, formatInteger} from '../utils/formatBytes';
import {SelectedObject} from '../state/types';
import {ErrorNotice} from './ErrorNotice';
import {LoadingOverlay} from './LoadingOverlay';
import './TableStatsView.css';

interface TableStatsViewProps {
    selected: SelectedObject | null;
    hasDatabase: boolean;
    onOpenDatabase: () => void;
}

interface StatRow {
    label: string;
    value: string;
    hint?: string;
}

export function TableStatsView({selected, hasDatabase, onOpenDatabase}: TableStatsViewProps) {
    const [stats, setStats] = useState<model.ObjectStats | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<APIError | null>(null);
    const requestRef = useRef(0);
    const showLoading = useDebouncedLoading(loading);

    const objectName =
        selected && (selected.kind === 'table' || selected.kind === 'view') ? selected.name : null;

    const loadStats = useCallback(async () => {
        const requestId = ++requestRef.current;
        if (!objectName) {
            setStats(null);
            return;
        }
        setLoading(true);
        setError(null);
        try {
            const resp = await WailsAPI.getObjectStats(objectName);
            if (requestId === requestRef.current) {
                setStats(resp);
            }
        } catch (err) {
            if (requestId === requestRef.current) {
                setError(formatAPIError(err));
                setStats(null);
            }
        } finally {
            if (requestId === requestRef.current) {
                setLoading(false);
            }
        }
    }, [objectName]);

    useEffect(() => {
        loadStats();
    }, [loadStats]);

    if (!hasDatabase) {
        return (
            <div className="panel-empty">
                <h2 className="panel-empty-title">No database open</h2>
                <p>Open a SQLite file to view table statistics.</p>
                <button type="button" className="btn btn-primary" onClick={onOpenDatabase}>
                    Open database
                </button>
            </div>
        );
    }

    if (!objectName) {
        return (
            <div className="table-stats">
                <p className="panel-hint">Select a table or view in the sidebar to see statistics.</p>
            </div>
        );
    }

    const rows: StatRow[] = stats
        ? [
              {label: 'Object type', value: stats.kind === 'view' ? 'View' : 'Table'},
              {label: 'Rows (exact)', value: formatInteger(stats.rowCount)},
              {
                  label: 'Rows (estimated)',
                  value:
                      stats.estimatedRowCount != null
                          ? formatInteger(stats.estimatedRowCount)
                          : '—',
                  hint: 'From sqlite_stat1 after ANALYZE',
              },
              {label: 'Columns', value: formatInteger(stats.columnCount)},
              {label: 'Indexes', value: formatInteger(stats.indexCount)},
              {label: 'Foreign keys', value: formatInteger(stats.foreignKeyCount)},
              {label: 'Triggers', value: formatInteger(stats.triggerCount)},
              {
                  label: 'Primary key',
                  value:
                      stats.primaryKeyColumns?.length > 0
                          ? stats.primaryKeyColumns.join(', ')
                          : '—',
              },
              {
                  label: 'Storage (table)',
                  value: stats.storageBytes != null ? formatBytes(stats.storageBytes) : '—',
                  hint: 'From dbstat virtual table when available',
              },
              {
                  label: 'WITHOUT ROWID',
                  value: stats.withoutRowId ? 'Yes' : 'No',
              },
              {
                  label: 'Database file size',
                  value: formatBytes(stats.databaseFileBytes),
              },
              {
                  label: 'Database pages',
                  value: `${formatInteger(stats.databasePageCount)} × ${formatBytes(stats.databasePageSize)}`,
                  hint: `Allocated ≈ ${formatBytes(stats.databaseUsedBytes)}`,
              },
              {
                  label: 'Query time',
                  value: `${stats.durationMs} ms`,
              },
          ]
        : [];

    return (
        <div className="table-stats panel-with-overlay">
            <div className="table-stats-toolbar">
                <h2 className="table-stats-title">{objectName}</h2>
                <button type="button" className="btn" onClick={loadStats} disabled={loading}>
                    Refresh
                </button>
            </div>

            <ErrorNotice error={error} className="table-stats-error" />

            <div className="table-stats-body">
                {showLoading && <LoadingOverlay label="Loading statistics..." />}
                {stats && (
                    <table className="table-stats-grid">
                        <tbody>
                            {rows.map((row) => (
                                <tr key={row.label}>
                                    <th scope="row">{row.label}</th>
                                    <td title={row.hint}>{row.value}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}
            </div>
        </div>
    );
}
