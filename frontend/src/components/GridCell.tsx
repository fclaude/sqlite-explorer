import {blobCellTooltip, formatCellDisplay, model} from '../api';
import {isCellExpandable} from '../utils/cellDetail';

interface GridCellProps {
    cell: model.CellValue;
    onOpen: () => void;
}

export function GridCell({cell, onOpen}: GridCellProps) {
    const display = formatCellDisplay(cell);
    const tooltip = blobCellTooltip(cell) ?? (isCellExpandable(cell) ? 'Click to view full value' : display);

    return (
        <button
            type="button"
            className={`grid-cell-btn${cell.kind === 'blob' ? ' cell-blob' : ''}`}
            title={tooltip}
            onClick={(e) => {
                e.stopPropagation();
                onOpen();
            }}
        >
            {display}
        </button>
    );
}
