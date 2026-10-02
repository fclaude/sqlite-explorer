import {model} from '../api';

interface StatementPermissionsProps {
    id?: string;
    categories: model.StatementCategory[];
    allowed: string[];
    readOnlyDatabase: boolean;
    onChange: (id: string, allowed: boolean) => void;
}

/** Lists which SQL statements the editor runs, and lets the user allow more categories. */
export function StatementPermissions({id, categories, allowed, readOnlyDatabase, onChange}: StatementPermissionsProps) {
    const read = categories.find((c) => c.id === 'read');
    const never = categories.find((c) => c.id === 'forbidden');
    const configurable = categories.filter((c) => c.configurable);

    return (
        <section id={id} className="sql-permissions" aria-label="Statement permissions">
            {read && (
                <p className="sql-permissions-line">
                    <strong>Always allowed:</strong> {read.statements.join(', ')}.{' '}
                    {read.description}
                    {read.pragmas && read.pragmas.length > 0 && (
                        <details className="sql-permissions-pragmas">
                            <summary>Read-only PRAGMAs</summary>
                            <code>{read.pragmas.join(', ')}</code>
                        </details>
                    )}
                </p>
            )}
            {readOnlyDatabase && (
                <p className="sql-permissions-line sql-permissions-warning">
                    The database is open read-only, so only read queries can run.
                </p>
            )}
            <fieldset className="sql-permissions-options" disabled={readOnlyDatabase}>
                <legend>Also allow</legend>
                {configurable.map((c) => (
                    <label key={c.id} className="sql-permissions-option">
                        <input
                            type="checkbox"
                            checked={allowed.includes(c.id)}
                            onChange={(e) => onChange(c.id, e.target.checked)}
                        />
                        <span>
                            <span className="sql-permissions-label">{c.label}</span>
                            <code className="sql-permissions-statements">{c.statements.join(', ')}</code>
                            <span className="sql-permissions-description">{c.description}</span>
                        </span>
                    </label>
                ))}
            </fieldset>
            {never && (
                <p className="sql-permissions-line">
                    <strong>{never.label}:</strong> {never.statements.join(', ')}. {never.description}
                </p>
            )}
            <p className="sql-permissions-line sql-permissions-hint">
                Statements that change anything run on their own connection, which is closed when the run ends.
                Export CSV only runs read queries.
            </p>
        </section>
    );
}
