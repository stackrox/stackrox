import React, { ReactElement } from 'react';
import { PageSection, Title, Divider } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import ClustersTable from './ClustersTable';

function ClustersListPage(): ReactElement {
    return (
        <>
            <PageTitle title="Clusters" />
            <PageSection variant="light">
                <Title headingLevel="h1">Clusters</Title>
            </PageSection>
            <Divider component="div" />
            <ClustersTable />
        </>
    );
}

export default ClustersListPage;
