import './SqlEditor.css';

interface SqlEditorProps {
    hasDatabase: boolean;
    onOpenDatabase: () => void;
}

export function SqlEditor({hasDatabase, onOpenDatabase}: SqlEditorProps) {
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
            <p className="panel-hint">SQL query runner will be available in Stage 6.</p>
            <textarea
                className="sql-textarea"
                placeholder="SELECT * FROM ..."
                disabled
                rows={12}
            />
        </div>
    );
}
