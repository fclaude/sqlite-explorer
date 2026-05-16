import {SelectedObject} from '../state/types';
import './DataGrid.css';

interface DataGridProps {
    selected: SelectedObject | null;
    hasDatabase: boolean;
    onOpenDatabase: () => void;
}

export function DataGrid({selected, hasDatabase, onOpenDatabase}: DataGridProps) {
    if (!hasDatabase) {
        return (
            <EmptyPanel
                title="No database open"
                message="Open a SQLite file to browse table data."
                onOpenDatabase={onOpenDatabase}
            />
        );
    }

    if (!selected || (selected.kind !== 'table' && selected.kind !== 'view')) {
        return (
            <div className="data-grid">
                <p className="panel-hint">Select a table or view in the sidebar to browse rows.</p>
            </div>
        );
    }

    return (
        <div className="data-grid">
            <p className="panel-hint">
                Data browser for <strong>{selected.name}</strong> will be available in Stage 5.
            </p>
        </div>
    );
}

function EmptyPanel({
    title,
    message,
    onOpenDatabase,
}: {
    title: string;
    message: string;
    onOpenDatabase: () => void;
}) {
    return (
        <div className="panel-empty">
            <h2 className="panel-empty-title">{title}</h2>
            <p>{message}</p>
            <button type="button" className="btn btn-primary" onClick={onOpenDatabase}>
                Open database
            </button>
        </div>
    );
}
