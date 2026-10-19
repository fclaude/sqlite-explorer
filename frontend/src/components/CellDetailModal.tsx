import {useCallback, useEffect, useRef, useState} from 'react';
import {APIError, formatAPIError} from '../api';
import {CellDetailContext, cellDetailText, cellTypeLabel, rowLabel} from '../utils/cellDetail';
import {fieldEncoding, hasFieldDraftChange, isFieldEditable} from '../utils/recordSave';
import {ErrorNotice} from './ErrorNotice';
import './DetailModal.css';

interface CellDetailModalProps {
    context: CellDetailContext | null;
    onClose: () => void;
    onViewRow?: () => void;
    onSave?: (draft: string) => Promise<void>;
}

export function CellDetailModal({context, onClose, onViewRow, onSave}: CellDetailModalProps) {
    const [draft, setDraft] = useState('');
    const [copied, setCopied] = useState(false);
    const [saving, setSaving] = useState(false);
    const [saveError, setSaveError] = useState<APIError | null>(null);
    const textareaRef = useRef<HTMLTextAreaElement>(null);

    useEffect(() => {
        if (!context) {
            return;
        }
        setDraft(cellDetailText(context.cell));
        setCopied(false);
        setSaveError(null);
        const t = window.setTimeout(() => textareaRef.current?.focus(), 0);
        return () => window.clearTimeout(t);
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

    const copyValue = useCallback(async () => {
        try {
            await navigator.clipboard.writeText(draft);
            setCopied(true);
            window.setTimeout(() => setCopied(false), 1500);
        } catch {
            textareaRef.current?.select();
            document.execCommand('copy');
            setCopied(true);
            window.setTimeout(() => setCopied(false), 1500);
        }
    }, [draft]);

    const handleSave = useCallback(async () => {
        if (!context || !onSave) {
            return;
        }
        setSaving(true);
        setSaveError(null);
        try {
            await onSave(draft);
            onClose();
        } catch (err) {
            setSaveError(formatAPIError(err));
        } finally {
            setSaving(false);
        }
    }, [context, draft, onSave, onClose]);

    if (!context) {
        return null;
    }

    const title =
        context.tableName != null
            ? `${context.tableName}.${context.columnName}`
            : context.columnName;

    const fieldEditable = isFieldEditable(context.cell);
    const meta = context.columnMeta?.find((c) => c.name === context.columnName);
    const hex = fieldEncoding(context.cell, meta) === 'hex';
    const canSave =
        context.editable &&
        fieldEditable &&
        !!onSave &&
        context.rowId != null &&
        hasFieldDraftChange(draft, context.cell);

    let hint: string;
    if (context.source === 'query') {
        hint = 'Query results are read-only.';
    } else if (!context.editable) {
        hint = 'Edit to copy or inspect. Use View full row to edit table rows.';
    } else if (!fieldEditable) {
        hint = "Only a preview of this BLOB is loaded, so it can't be edited here.";
    } else if (hex) {
        hint = 'Edit the bytes as hex (for example 0x00ff) and click Save to update only this column.';
    } else {
        hint = 'Edit this field and click Save to update only this column.';
    }

    return (
        <div className="detail-modal-backdrop" onClick={onClose} role="presentation">
            <div
                className="detail-modal"
                role="dialog"
                aria-labelledby="cell-detail-title"
                aria-modal="true"
                onClick={(e) => e.stopPropagation()}
            >
                <header className="detail-modal-header">
                    <div>
                        <h2 id="cell-detail-title" className="detail-modal-title">
                            {title}
                        </h2>
                        <p className="detail-modal-meta">
                            {rowLabel(context)} · {cellTypeLabel(context.cell)}
                            {context.source === 'query' ? ' · query result' : ''}
                            {context.rowId != null ? ` · rowid ${context.rowId}` : ''}
                        </p>
                    </div>
                    <button type="button" className="detail-modal-close" onClick={onClose} aria-label="Close">
                        ×
                    </button>
                </header>

                <p className="detail-modal-hint">{hint}</p>
                <ErrorNotice error={saveError} className="detail-modal-error" />

                <textarea
                    ref={textareaRef}
                    className="detail-modal-editor"
                    value={draft}
                    onChange={(e) => setDraft(e.target.value)}
                    spellCheck={false}
                    disabled={saving}
                    readOnly={context.editable && !fieldEditable}
                    placeholder={context.cell.kind === 'null' ? 'NULL' : undefined}
                />

                <footer className="detail-modal-footer">
                    <button type="button" className="btn" onClick={copyValue} disabled={saving}>
                        {copied ? 'Copied' : 'Copy'}
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
                    {onViewRow && (
                        <button type="button" className="btn" onClick={onViewRow} disabled={saving}>
                            View full row
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
