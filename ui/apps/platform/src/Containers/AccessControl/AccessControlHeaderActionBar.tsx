import React, { ReactElement, ReactNode } from 'react';
import { Divider, PageSection, Split, SplitItem } from '@patternfly/react-core';

export type AccessControlHeaderActionBarProps = {
    /**
     * The display level or description component to render.
     * This component will fill the available space on the left side of the bar.
     */
    displayComponent: ReactNode;
    /**
     * The UI component that performs the main action on this page.
     */
    actionComponent?: ReactNode;
};

/**
 * Renders a display item, usually a description, and a main action UI item for the user's
 * primary action on this page.
 */
function AccessControlHeaderActionBar({
    displayComponent,
    actionComponent,
}: AccessControlHeaderActionBarProps): ReactElement {
    return (
        <>
            <PageSection variant="light" className="pf-u-py-md">
                <Split>
                    <SplitItem isFilled>{displayComponent}</SplitItem>
                    {actionComponent && <SplitItem>{actionComponent}</SplitItem>}
                </Split>
            </PageSection>
            <Divider component="div" />
        </>
    );
}

export default AccessControlHeaderActionBar;
