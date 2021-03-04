import React, { ReactElement, ReactNode } from 'react';

export type SidePanelAdjacentAreaProps = {
    children: ReactNode;
    isWider?: boolean;
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
    isWider = false,
}: SidePanelAdjacentAreaProps): ReactElement {
    return (
        <div
            className={`bg-base-100 border-base-400 border-l flex-shrink-0 h-full absolute left-0 top-0 w-full z-10 md:relative ${
                isWider ? 'md:w-3/5 lg:w-1/2 xl:w-2/5 xxl:w-1/3' : 'md:w-2/5 xl:w-1/3 xxl:w-1/4'
            }`}
        >
            {children}
        </div>
    );
}

export default SidePanelAdjacentArea;
