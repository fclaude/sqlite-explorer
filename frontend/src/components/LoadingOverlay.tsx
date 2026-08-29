import './LoadingOverlay.css';

interface LoadingOverlayProps {
    label?: string;
}

export function LoadingOverlay({label = 'Loading...'}: LoadingOverlayProps) {
    return (
        <div className="loading-overlay" role="status" aria-live="polite">
            <span className="loading-spinner" aria-hidden="true" />
            <span className="loading-label">{label}</span>
        </div>
    );
}
