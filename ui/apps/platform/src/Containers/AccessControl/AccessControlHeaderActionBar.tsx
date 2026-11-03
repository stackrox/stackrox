import type { ReactElement, ReactNode } from 'react';
import { Divider, Flex, FlexItem, PageSection } from '@patternfly/react-core';

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
    inviteComponent?: ReactNode;
};

/**
 * Renders a display item, usually a description, and a main action UI item for the user's
 * primary action on this page.
 */
function AccessControlHeaderActionBar({
    displayComponent,
    actionComponent,
    inviteComponent,
}: AccessControlHeaderActionBarProps): ReactElement {
    return (
        <>
            <PageSection>
                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem grow={{ default: 'grow' }}>{displayComponent}</FlexItem>
                    {inviteComponent && <FlexItem>{inviteComponent}</FlexItem>}
                    {actionComponent && <FlexItem>{actionComponent}</FlexItem>}
                </Flex>
            </PageSection>
            <Divider component="div" />
        </>
    );
}

export default AccessControlHeaderActionBar;
