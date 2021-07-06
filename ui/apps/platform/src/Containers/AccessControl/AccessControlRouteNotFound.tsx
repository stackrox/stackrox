import React, { CSSProperties, ReactElement } from 'react';
import { Alert, AlertVariant, List, ListItem, Stack, StackItem } from '@patternfly/react-core';

import AccessControlHeading from './AccessControlHeading';
import AccessControlNav from './AccessControlNav';

// Separate list from the title with same margin-top as second list item from the first.
const styleList = {
    marginTop: 'var(--pf-c-list--li--MarginTop)',
} as CSSProperties;

function AccessControlRouteNotFound(): ReactElement {
    return (
        <Stack hasGutter>
            <StackItem>
                <AccessControlHeading />
                <AccessControlNav />
            </StackItem>
            <StackItem>
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
            </StackItem>
        </Stack>
    );
}

export default AccessControlRouteNotFound;
