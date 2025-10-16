import React from 'react';
import type { ReactElement, ReactNode } from 'react';

type Width = '1/3' | '2/5' | '1/2' | '3/5';

export type SidePanelAdjacentAreaProps = {
    children: ReactNode;
    width?: Width;
};

const widthClassNames: Record<Width, string> = {
    '1/3': 'md:w-1/3', // Compliance
    '2/5': 'md:w-2/5 xl:w-1/3 xxl:w-1/4', // Violations
    '1/2': 'md:w-1/2', // Integrations
    '3/5': 'md:w-3/5 lg:w-1/2 xl:w-2/5 xxl:w-1/3', // Risk
};

/*
 * Render an area (without animation nor overlay) that contains the content of a side panel.
 * Assume its parent has position relative.
 * Assume it follows its main panel sibling.
 *
 * If page width is less than medium, the side panel completely covers the main panel.
 * If page width is at least medium, the side panel has a fraction of page width (not including side nav).
 */
function SidePanelAdjacentArea({
    children,
    width = '1/2',
}: SidePanelAdjacentAreaProps): ReactElement {
    return (
        <div
            className={`bg-base-100 border-base-400 border-l flex-shrink-0 h-full absolute left-0 top-0 w-full z-10 md:relative ${widthClassNames[width]}`}
        >
            {children}
        </div>
    );
}

export default SidePanelAdjacentArea;
