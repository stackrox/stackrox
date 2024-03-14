import React from 'react';
import { PageSection, Breadcrumb, Divider, BreadcrumbItem } from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { getOverviewCvesPath } from '../utils/searchUtils';

const workloadCveOverviewCvePath = getOverviewCvesPath({
    entityTab: 'CVE',
});

function NodeCvePage() {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { cveId } = useParams() as { cveId: string };

    const nodeCveName = cveId; // TODO Replace me with queried data

    return (
        <>
            <PageTitle title={`Node CVEs - NodeCVE ${nodeCveName}`} />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={workloadCveOverviewCvePath}>CVEs</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{nodeCveName} </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light"></PageSection>
        </>
    );
}

export default NodeCvePage;
