import React, { CSSProperties, ReactElement } from 'react';
import { Alert, AlertVariant, List, ListItem, PageSection } from '@patternfly/react-core';

import AccessControlHeading from './AccessControlHeading';

// Separate list from the title with same margin-top as second list item from the first.
const styleList = {
    marginTop: 'var(--pf-c-list--li--MarginTop)',
} as CSSProperties;

function AccessControlRouteNotFound(): ReactElement {
    return (
        <>
            <AccessControlHeading />
            <PageSection variant="light">
                <Alert
                    title="Access Control route not found"
                    variant={AlertVariant.warning}
                    isInline
                >
                    <List style={styleList}>
                        <ListItem>Click the browser Back button</ListItem>
                        <ListItem>Click a tab under Access Control</ListItem>
                    </List>
                </Alert>
            </PageSection>
        </>
    );
}

export default AccessControlRouteNotFound;
