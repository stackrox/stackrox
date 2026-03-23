import type { ReactElement } from 'react';
import { Alert, List, ListItem, PageSection } from '@patternfly/react-core';

import AccessControlHeading from './AccessControlHeading';

function AccessControlRouteNotFound(): ReactElement {
    return (
        <>
            <AccessControlHeading />
            <PageSection>
                <Alert
                    title="Access Control route not found"
                    component="p"
                    variant="warning"
                    isInline
                >
                    <List>
                        <ListItem>Click the browser Back button</ListItem>
                        <ListItem>Click a tab under Access Control</ListItem>
                    </List>
                </Alert>
            </PageSection>
        </>
    );
}

export default AccessControlRouteNotFound;
