import {useCallback, useEffect, useState} from 'react';
import {APIError, formatAPIError, model} from '../api';
import {cellTypeLabel, RecordRowContext, rowLabel} from '../utils/cellDetail';
import {fieldDraft, fieldEncoding, hasDraftChanges, isFieldEditable} from '../utils/recordSave';
import {ErrorNotice} from './ErrorNotice';
import './DetailModal.css';

interface RecordDetailModalProps {
    context: RecordRowContext | null;
    onClose: () => void;
    onOpenCell?: (columnIndex: number) => void;
    onSave?: (drafts: Record<string, string>) => Promise<void>;
}

export function RecordDetailModal({context, onClose, onOpenCell, onSave}: RecordDetailModalProps) {
    const [drafts, setDrafts] = useState<Record<string, string>>({});
    const [saving, setSaving] = useState(false);
    const [saveError, setSaveError] = useState<APIError | null>(null);

    useEffect(() => {
        if (!context) {
            setDrafts({});
            setSaveError(null);
            return;
        }
        const next: Record<string, string> = {};
        context.columns.forEach((col, i) => {
            next[col.name] = fieldDraft(context.cells[i]);
        });
        setDrafts(next);
        setSaveError(null);
    }, [context]);

    useEffect(() => {
        if (!context) {
            return;
        }
        const onKey = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                onClose();
            }
        };
        window.addEventListener('keydown', onKey);
        return () => window.removeEventListener('keydown', onKey);
    }, [context, onClose]);

    const copyAll = useCallback(async () => {
        if (!context) {
            return;
        }
        const lines = context.columns.map((col) => {
            const v = drafts[col.name] ?? '';
            return `${col.name}\t${v}`;
        });
        const text = lines.join('\n');
        try {
            await navigator.clipboard.writeText(text);
        } catch {
            // ignore
        }
    }, [context, drafts]);

    const handleSave = useCallback(async () => {
        if (!context || !onSave) {
            return;
        }
        setSaving(true);
        setSaveError(null);
        try {
            await onSave(drafts);
            onClose();
        } catch (err) {
            setSaveError(formatAPIError(err));
        } finally {
            setSaving(false);
        }
    }, [context, drafts, onSave, onClose]);

    if (!context) {
        return null;
    }

    const title =
        context.tableName != null ? `${context.tableName} — ${rowLabel(context)}` : rowLabel(context);

    const canSave =
        context.editable &&
        !!onSave &&
        context.rowId != null &&
        hasDraftChanges(context.columns, context.cells, drafts);

    const hint =
        context.editable && context.source === 'table'
            ? 'Edit fields and click Save to update this row in the database.'
            : context.source === 'query'
              ? 'Query results are read-only. Open the table in the Data tab to edit rows.'
              : 'This row cannot be edited.';

    return (
        <div className="detail-modal-backdrop" onClick={onClose} role="presentation">
            <div
                className="detail-modal detail-modal-record"
                role="dialog"
                aria-labelledby="record-detail-title"
                aria-modal="true"
                onClick={(e) => e.stopPropagation()}
            >
                <header className="detail-modal-header">
                    <div>
                        <h2 id="record-detail-title" className="detail-modal-title">
                            {title}
                        </h2>
                        <p className="detail-modal-meta">
                            {context.columns.length} fields
                            {context.source === 'query' ? ' · query result' : ' · table browse'}
                            {context.rowId != null ? ` · rowid ${context.rowId}` : ''}
                        </p>
                    </div>
                    <button type="button" className="detail-modal-close" onClick={onClose} aria-label="Close">
                        ×
                    </button>
                </header>

                <p className="detail-modal-hint">{hint}</p>
                <ErrorNotice error={saveError} className="detail-modal-error" />

                <div className="record-detail-fields">
                    {context.columns.map((col, i) => {
                        const cell = context.cells[i];
                        const meta = context.columnMeta?.find((c) => c.name === col.name);
                        const isPk = (meta?.primaryKey ?? 0) > 0;
                        const editable = isFieldEditable(cell);
                        const hex = fieldEncoding(cell, meta) === 'hex';
                        return (
                            <div key={col.name} className="record-detail-field">
                                <div className="record-detail-field-head">
                                    <label className="record-detail-label" htmlFor={`field-${col.name}`}>
                                        {col.name}
                                        {isPk ? ' (PK)' : ''}
                                    </label>
                                    <span className="record-detail-type">
                                        {cellTypeLabel(cell)}
                                        {hex && context.editable ? ' · hex' : ''}
                                    </span>
                                    {onOpenCell && (
                                        <button
                                            type="button"
                                            className="record-detail-expand"
                                            onClick={() => onOpenCell(i)}
                                        >
                                            Expand
                                        </button>
                                    )}
                                </div>
                                <textarea
                                    id={`field-${col.name}`}
                                    className="record-detail-input"
                                    value={drafts[col.name] ?? ''}
                                    onChange={(e) =>
                                        setDrafts((prev) => ({...prev, [col.name]: e.target.value}))
                                    }
                                    disabled={!context.editable || saving || !editable}
                                    placeholder={cell.kind === 'null' ? (hex ? 'NULL (enter hex bytes, e.g. 0x00ff)' : 'NULL') : undefined}
                                    rows={Math.min(12, Math.max(2, Math.ceil((drafts[col.name]?.length ?? 0) / 80)))}
                                    spellCheck={false}
                                />
                                {!editable && context.editable && (
                                    <p className="record-detail-note">
                                        Only a preview of this BLOB is loaded, so it can't be edited here.
                                    </p>
                                )}
                            </div>
                        );
                    })}
                </div>

                <footer className="detail-modal-footer">
                    <button type="button" className="btn" onClick={copyAll} disabled={saving}>
                        Copy row
                    </button>
                    {context.editable && onSave && (
                        <button
                            type="button"
                            className="btn btn-primary"
                            onClick={handleSave}
                            disabled={!canSave || saving}
                        >
                            {saving ? 'Saving...' : 'Save'}
                        </button>
                    )}
                    <button type="button" className="btn" onClick={onClose} disabled={saving}>
                        Close
                    </button>
                </footer>
            </div>
        </div>
    );
}
