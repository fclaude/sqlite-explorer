import {useState} from 'react';
import './App.css';
import {CloseDatabase, DatabaseInfo, GetSchema, OpenDatabase} from "../wailsjs/go/backend/App";
import {model} from "../wailsjs/go/models";

function App() {
    const [dbInfo, setDbInfo] = useState<model.DatabaseInfo | null>(null);
    const [message, setMessage] = useState('Open a SQLite database to begin.');
    const [schemaSummary, setSchemaSummary] = useState('');

    async function openDatabase() {
        try {
            const info: model.DatabaseInfo = await OpenDatabase();
            setDbInfo(info);
            setMessage(`Opened: ${info.path} (${info.sizeBytes} bytes, read-only: ${info.readOnly})`);
            setSchemaSummary('');
        } catch (err) {
            setDbInfo(null);
            setSchemaSummary('');
            setMessage(formatError(err));
        }
    }

    async function loadSchema() {
        try {
            const schema: model.SchemaInfo = await GetSchema();
            setSchemaSummary(
                `Schema: ${schema.tables?.length ?? 0} tables, ` +
                `${schema.views?.length ?? 0} views, ` +
                `${schema.indexes?.length ?? 0} indexes, ` +
                `${schema.triggers?.length ?? 0} triggers`
            );
        } catch (err) {
            setSchemaSummary(formatError(err));
        }
    }

    async function closeDatabase() {
        try {
            await CloseDatabase();
            setDbInfo(null);
            setSchemaSummary('');
            setMessage('Database closed.');
        } catch (err) {
            setMessage(formatError(err));
        }
    }

    async function refreshInfo() {
        if (!dbInfo) {
            return;
        }
        try {
            const info: model.DatabaseInfo = await DatabaseInfo();
            setDbInfo(info);
            setMessage(`Opened: ${info.path} (${info.sizeBytes} bytes, read-only: ${info.readOnly})`);
        } catch (err) {
            setMessage(formatError(err));
        }
    }

    return (
        <div id="App" className="app-root">
            <header className="toolbar">
                <h1>SQLite Explorer</h1>
                <div className="toolbar-actions">
                    <button className="btn" onClick={openDatabase}>Open database</button>
                    <button className="btn" onClick={closeDatabase} disabled={!dbInfo}>Close</button>
                    <button className="btn" onClick={refreshInfo} disabled={!dbInfo}>Refresh info</button>
                    <button className="btn" onClick={loadSchema} disabled={!dbInfo}>Load schema</button>
                </div>
            </header>
            <main className="content">
                <p className="message">{message}</p>
                {schemaSummary && <p className="schema-summary">{schemaSummary}</p>}
                {dbInfo && (
                    <dl className="db-info">
                        <dt>Path</dt>
                        <dd>{dbInfo.path}</dd>
                        <dt>Size</dt>
                        <dd>{dbInfo.sizeBytes} bytes</dd>
                        <dt>Read-only</dt>
                        <dd>{dbInfo.readOnly ? 'yes' : 'no'}</dd>
                    </dl>
                )}
            </main>
        </div>
    );
}

function formatError(err: unknown): string {
    if (err instanceof Error) {
        return err.message;
    }
    return String(err);
}

export default App
