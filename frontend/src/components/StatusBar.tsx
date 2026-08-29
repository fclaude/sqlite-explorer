import React, {useState} from 'react';
import {model} from '../api';
import {StatusState} from '../state/types';
import './StatusBar.css';

const TRUNCATION_DOCS =
    'https://www.sqlite.org/limits.html — result sets are capped at 1000 rows for responsiveness.';

interface StatusBarProps {
    dbInfo: model.DatabaseInfo | null;
    status: StatusState;
    onClearError: () => void;
}

export function StatusBar({dbInfo, status, onClearError}: StatusBarProps) {
    const [showErrorDetail, setShowErrorDetail] = useState(false);
    const dbLabel = dbInfo?.path ?? 'No database';

    return (
        <footer className="status-bar">
            <span className="status-segment status-db" title={dbLabel}>
                {dbLabel}
            </span>
            <span className="status-segment status-duration">
                {status.queryDurationMs != null ? `Query: ${status.queryDurationMs}ms` : 'Query: --'}
            </span>
            <span className="status-segment status-page">
                {status.pageInfo ?? 'Page: --'}
            </span>
            {status.truncated && (
                <a
                    className="status-segment status-truncated"
                    href={TRUNCATION_DOCS}
                    target="_blank"
                    rel="noopener noreferrer"
                    title={TRUNCATION_DOCS}
                >
                    Showing first 1000 rows
                </a>
            )}
            <span className="status-segment status-error">
                {status.lastError ? (
                    <>
                        <button
                            type="button"
                            className="status-error-btn"
                            onClick={() => setShowErrorDetail((v) => !v)}
                            title={status.lastErrorDetail ?? undefined}
                        >
                            Error: {status.lastError}
                        </button>
                        <button type="button" className="status-error-dismiss" onClick={onClearError}>
                            x
                        </button>
                    </>
                ) : (
                    <span className="status-ok">No errors</span>
                )}
            </span>
            {showErrorDetail && status.lastErrorDetail && (
                <pre className="status-error-detail">{status.lastErrorDetail}</pre>
            )}
            {status.loading && <span className="status-loading">Loading...</span>}
        </footer>
    );
}
