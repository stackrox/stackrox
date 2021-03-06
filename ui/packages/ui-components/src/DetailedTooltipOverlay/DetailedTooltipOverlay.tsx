import React, { ReactElement, ReactNode } from 'react';

export type DetailedTooltipOverlayProps = {
    title: string;
    subtitle?: string;
    body: ReactNode;
    footer?: ReactNode;
    extraClassName?: string;
};

/**
 * Alternative to {@link TooltipOverlay} that provides layout for complex tooltip content with
 * title, subtitle and footer in addition to the main body.
 */
function DetailedTooltipOverlay({
    title,
    subtitle,
    body,
    footer,
    extraClassName = '',
}: DetailedTooltipOverlayProps): ReactElement | null {
    if (!title || !body) {
        return null;
    }

    return (
        <div
            className={`rox-tooltip-overlay min-w-32 max-h-100 flex flex-col flex-1 ${extraClassName}`}
        >
            <div className="text-left flex flex-col border-b leading-normal py-1 px-2 detailed-overlay-header">
                <h1 className="font-700 text-lg" data-testid="tooltip-title">
                    {title}
                </h1>
                {subtitle && <span data-testid="tooltip-subtitle">{subtitle}</span>}
            </div>
            <div
                className="flex flex-1 flex-col text-left overflow-auto p-2 text-sm"
                data-testid="tooltip-body"
            >
                {body}
            </div>
            {!!footer && (
                <footer
                    className="p-2 font-700 text-left text-sm leading-loose"
                    data-testid="tooltip-footer"
                >
                    {footer}
                </footer>
            )}
        </div>
    );
}

export default DetailedTooltipOverlay;
