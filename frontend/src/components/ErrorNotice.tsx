import {ReactNode} from 'react';
import {APIError} from '../api';
import './ErrorNotice.css';

interface ErrorNoticeProps {
    error: Pick<APIError, 'message' | 'detail'> | null;
    className?: string;
    children?: ReactNode;
}

/** Shows an error message with SQLite's own detail (for example the syntax error position). */
export function ErrorNotice({error, className, children}: ErrorNoticeProps) {
    if (!error) {
        return null;
    }
    return (
        <div className={`error-notice${className ? ` ${className}` : ''}`} role="alert">
            <p className="error-notice-message">{error.message}</p>
            {error.detail && <pre className="error-notice-detail">{error.detail}</pre>}
            {children}
        </div>
    );
}
