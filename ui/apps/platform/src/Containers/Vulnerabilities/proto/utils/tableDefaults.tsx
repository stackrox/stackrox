import React from 'react';

// Column width constants (in pixels)
export const CVE_NAME_WIDTH = 180;
export const SEVERITY_WIDTH = 120;
export const COMPONENT_NAME_WIDTH = 400; // max-width
export const IMAGE_DIGEST_WIDTH = 300; // max-width
export const CVSS_SCORE_WIDTH = 80;
export const COUNT_WIDTH = 80;
export const DATE_WIDTH = 150;

// Table styling constants
export const TABLE_HEADER_STYLE: React.CSSProperties = {
    padding: '10px 12px',
    fontWeight: 600,
    fontSize: '11px',
    textTransform: 'uppercase',
    letterSpacing: '0.5px',
};

export const TABLE_CELL_STYLE: React.CSSProperties = {
    padding: '12px',
};

/**
 * Format ISO timestamp to readable date
 * @param isoString - ISO 8601 timestamp
 * @returns Formatted date like "Jun 1, 2026"
 */
export function formatDate(isoString: string | null | undefined): string {
    if (!isoString) return '–';

    try {
        const date = new Date(isoString);
        return new Intl.DateTimeFormat('en-US', {
            month: 'short',
            day: 'numeric',
            year: 'numeric',
        }).format(date);
    } catch {
        return '–';
    }
}

/**
 * Truncate text with ellipsis and show full text on hover
 * @param text - Text to truncate
 * @param maxWidth - Maximum width in pixels
 * @returns JSX element with truncation
 */
export function truncateWithTooltip(
    text: string | null | undefined,
    maxWidth: number
): JSX.Element {
    if (!text) {
        return <span>–</span>;
    }

    // Only apply truncation if text is reasonably long
    if (text.length < 50) {
        return <span>{text}</span>;
    }

    const style: React.CSSProperties = {
        maxWidth: `${maxWidth}px`,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        display: 'inline-block',
    };

    return (
        <span style={style} title={text}>
            {text}
        </span>
    );
}
