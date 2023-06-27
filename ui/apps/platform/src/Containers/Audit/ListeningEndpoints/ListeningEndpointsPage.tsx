import React from 'react';
import { Divider, PageSection, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';

function ListeningEndpointsPage() {
    return (
        <>
            <PageTitle title="Listening Endpoints" />
            <PageSection variant="light">
                <Title headingLevel="h1">Listening endpoints</Title>
            </PageSection>
            <Divider component="div" />
            <PageSection />
        </>
    );
}

export default ListeningEndpointsPage;
