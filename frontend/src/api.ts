import {
    CloseDatabase,
    DatabaseInfo,
    GetSchema,
    OpenDatabase,
} from '../wailsjs/go/backend/App';
import {model} from '../wailsjs/go/models';

/** Typed wrapper over Wails-generated bindings. Components must use this, not wailsjs directly. */
export const WailsAPI = {
    openDatabase(): Promise<model.DatabaseInfo> {
        return OpenDatabase();
    },

    closeDatabase(): Promise<void> {
        return CloseDatabase();
    },

    databaseInfo(): Promise<model.DatabaseInfo> {
        return DatabaseInfo();
    },

    getSchema(): Promise<model.SchemaInfo> {
        return GetSchema();
    },
};

export type {model};

export function formatAPIError(err: unknown): {message: string; detail: string} {
    if (err instanceof Error) {
        return {message: err.message, detail: err.stack ?? ''};
    }
    return {message: String(err), detail: ''};
}
