import React from 'react';
import { Tooltip } from '@patternfly/react-core';

/*
 * PageBody is sibling of PageHeader and parent of main and side panels.
 */
export function PageBody({ children }) {
    return <div className="flex flex-1 h-full relative z-0">{children}</div>;
}

/*
 * PanelNew is parent of PanelHead and PanelBody.
 */
export function PanelNew({ children, testid }) {
    return (
        <div className="flex flex-col flex-1 h-full w-full" data-testid={testid}>
            {children}
        </div>
    );
}

/*
 * PanelHead is parent of the following:
 * PanelTitle or entity-specific component like EntityBreadCrumbs
 * PanelHeadEnd, which has flex end alignment
 */
export function PanelHead({ children }) {
    return <div className="border-base-400 border-b flex h-14 w-full">{children}</div>;
}

export function PanelTitle({ testid = '', breakAll = true, text }) {
    return (
        <div
            className="flex items-center leading-normal min-w-24 overflow-hidden px-4 text-base-600"
            data-testid={testid || null}
        >
            <Tooltip content={text}>
                <h2>
                    <div className={`font-700 line-clamp ${breakAll ? 'break-all' : ''}`}>
                        {text}
                    </div>
                </h2>
            </Tooltip>
        </div>
    );
}

/*
 * PanelHeadStart is a parent of multiple components at the start of the panel head.
 * That is, instead of PageTitle.
 */
export function PanelHeadStart({ children, testid = null }) {
    return (
        <div className="flex" data-testid={testid}>
            {children}
        </div>
    );
}

/*
 * PanelHeadEnd, which has flex end alignment, is parent of reusable components.
 * main panel: search filter, table pagination
 * side panel: external link button, close button
 */
export function PanelHeadEnd({ children }) {
    return <div className="flex flex-1 items-center justify-end pl-3 relative">{children}</div>;
}

/*
 * PanelBody is parent of one or more entity-specific components.
 */
export function PanelBody({ children }) {
    return <div className="h-full overflow-y-auto">{children}</div>;
}

export const headerClassName = 'flex w-full min-h-14 border-b border-base-400';
