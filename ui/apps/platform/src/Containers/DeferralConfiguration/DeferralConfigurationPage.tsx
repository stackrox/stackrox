import React from 'react';
import { PageSection, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';

function DeferralConfigurationPage() {
    return (
        <>
            <PageTitle title="Deferral configuration" />
            <PageSection variant="light">
                <Title headingLevel="h1">Deferral configuration</Title>
            </PageSection>
        </>
    );
}

export default DeferralConfigurationPage;
