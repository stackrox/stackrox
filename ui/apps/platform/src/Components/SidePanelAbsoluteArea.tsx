import React from 'react';
import type { ReactElement, ReactNode } from 'react';

export type SidePanelAbsoluteAreaProps = {
    children: ReactNode;
};

/*
 * Render an area that contains the content of a side panel.
 * Assume its parent has position relative.
 * Assume it follows its main panel sibling.
 * A semi-transparent gray background color covers the main panel (underlay style).
 */
function SidePanelAbsoluteArea({ children }: SidePanelAbsoluteAreaProps): ReactElement {
    return (
        <div
            className="absolute flex h-full justify-end left-0 top-0 w-full z-10"
            style={{ backgroundColor: 'rgba(3, 3, 3, 0.62)' }}
        >
            <div className="bg-base-200 border-base-400 border-l h-full rounded-tl-lg shadow-sidepanel w-full lg:w-9/10">
                {children}
            </div>
        </div>
    );
}

export default SidePanelAbsoluteArea;
