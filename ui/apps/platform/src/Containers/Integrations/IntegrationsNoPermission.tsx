import React, { ReactElement } from 'react';
import {
    Alert,
    AlertVariant,
    Divider,
    PageSection,
    PageSectionVariants,
    Title,
} from '@patternfly/react-core';

function IntegrationsNoPermission(): ReactElement {
    return (
        <>
            <PageSection variant="light">
                <Title headingLevel="h1">Integrations</Title>
            </PageSection>
            <Divider component="div" />
            <PageSection variant={PageSectionVariants.light}>
                <Alert
                    className="pf-u-mt-md"
                    title="You do not have permission to view Integrations"
                    variant={AlertVariant.info}
                    isInline
                />
            </PageSection>
        </>
    );
}

export default IntegrationsNoPermission;
