import {useState} from 'react';
import './App.css';
import {CloseDatabase, DatabaseInfo, OpenDatabase} from "../wailsjs/go/backend/App";
import {model} from "../wailsjs/go/models";

function App() {
    const [dbInfo, setDbInfo] = useState<model.DatabaseInfo | null>(null);
    const [message, setMessage] = useState('Open a SQLite database to begin.');

    async function openDatabase() {
        try {
            const info: model.DatabaseInfo = await OpenDatabase();
            setDbInfo(info);
            setMessage(`Opened: ${info.path} (${info.sizeBytes} bytes, read-only: ${info.readOnly})`);
        } catch (err) {
            setDbInfo(null);
            setMessage(formatError(err));
        }
    }

    async function closeDatabase() {
        try {
            await CloseDatabase();
            setDbInfo(null);
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
                </div>
            </header>
            <main className="content">
                <p className="message">{message}</p>
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
