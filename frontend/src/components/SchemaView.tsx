import {model} from '../api';
import {SelectedObject} from '../state/types';
import './SchemaView.css';

interface SchemaViewProps {
    schema: model.SchemaInfo | null;
    selected: SelectedObject | null;
    hasDatabase: boolean;
    onOpenDatabase: () => void;
}

export function SchemaView({schema, selected, hasDatabase, onOpenDatabase}: SchemaViewProps) {
    if (!hasDatabase) {
        return (
            <div className="panel-empty">
                <h2 className="panel-empty-title">No database open</h2>
                <p>Open a SQLite file to inspect schema details.</p>
                <button type="button" className="btn btn-primary" onClick={onOpenDatabase}>
                    Open database
                </button>
            </div>
        );
    }

    if (!schema) {
        return <p className="panel-hint">Loading schema...</p>;
    }

    if (!selected) {
        return (
            <div className="schema-view">
                <p className="panel-hint">Select a table, view, index, or trigger in the sidebar.</p>
            </div>
        );
    }

    if (selected.kind === 'table') {
        const table = schema.tables?.find((t) => t.name === selected.name);
        if (!table) {
            return <p className="panel-hint">Table not found.</p>;
        }
        return (
            <div className="schema-view">
                <h2 className="panel-title">{table.name}</h2>
                <p className="panel-meta">Table</p>
                {table.sql && <pre className="schema-sql">{table.sql}</pre>}
                <h3>Columns</h3>
                <ColumnTable columns={table.columns} />
                {table.foreignKeys?.length > 0 && (
                    <>
                        <h3>Foreign keys</h3>
                        <ul className="schema-list">
                            {table.foreignKeys.map((fk) => (
                                <li key={`${fk.id}-${fk.seq}`}>
                                    {fk.from} references {fk.table}.{fk.to}
                                    {fk.onDelete ? ` ON DELETE ${fk.onDelete}` : ''}
                                </li>
                            ))}
                        </ul>
                    </>
                )}
            </div>
        );
    }

    if (selected.kind === 'view') {
        const view = schema.views?.find((v) => v.name === selected.name);
        if (!view) {
            return <p className="panel-hint">View not found.</p>;
        }
        return (
            <div className="schema-view">
                <h2 className="panel-title">{view.name}</h2>
                <p className="panel-meta">View</p>
                {view.sql && <pre className="schema-sql">{view.sql}</pre>}
                <h3>Columns</h3>
                <ColumnTable columns={view.columns} />
            </div>
        );
    }

    if (selected.kind === 'index') {
        const index = schema.indexes?.find((i) => i.name === selected.name);
        if (!index) {
            return <p className="panel-hint">Index not found.</p>;
        }
        return (
            <div className="schema-view">
                <h2 className="panel-title">{index.name}</h2>
                <p className="panel-meta">
                    Index on {index.table}
                    {index.unique ? ' (unique)' : ''}
                </p>
                {index.sql && <pre className="schema-sql">{index.sql}</pre>}
                <p>Columns: {index.columns?.map((c) => c.name).join(', ') || '(none)'}</p>
            </div>
        );
    }

    const trigger = schema.triggers?.find((t) => t.name === selected.name);
    if (!trigger) {
        return <p className="panel-hint">Trigger not found.</p>;
    }
    return (
        <div className="schema-view">
            <h2 className="panel-title">{trigger.name}</h2>
            <p className="panel-meta">Trigger on {trigger.table}</p>
            {trigger.sql && <pre className="schema-sql">{trigger.sql}</pre>}
        </div>
    );
}

function ColumnTable({columns}: {columns: model.ColumnInfo[]}) {
    if (!columns?.length) {
        return <p className="panel-hint">No columns.</p>;
    }
    return (
        <table className="schema-table">
            <thead>
                <tr>
                    <th>Name</th>
                    <th>Type</th>
                    <th>Null</th>
                    <th>Default</th>
                    <th>PK</th>
                </tr>
            </thead>
            <tbody>
                {columns.map((col) => (
                    <tr key={col.name}>
                        <td>{col.name}</td>
                        <td>{col.type || '—'}</td>
                        <td>{col.notNull ? 'NO' : 'YES'}</td>
                        <td>{col.defaultValue ?? '—'}</td>
                        <td>{col.primaryKey || '—'}</td>
                    </tr>
                ))}
            </tbody>
        </table>
    );
}
