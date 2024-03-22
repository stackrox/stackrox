import React from 'react';
import { PageSection, Breadcrumb, Divider, BreadcrumbItem, Skeleton } from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { getOverviewCvesPath } from '../utils/searchUtils';

const workloadCveOverviewCvePath = getOverviewCvesPath({
    entityTab: 'Node',
});

function NodePage() {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { nodeId } = useParams() as { nodeId: string };

    const nodeName: string | undefined = 'TODO';

    return (
        <>
            <PageTitle title={`Node CVEs - Node ${nodeName}`} />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={workloadCveOverviewCvePath}>Nodes</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {nodeName ?? (
                            <Skeleton screenreaderText="Loading Node name" width="200px" />
                        )}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
        </>
    );
}

export default NodePage;
