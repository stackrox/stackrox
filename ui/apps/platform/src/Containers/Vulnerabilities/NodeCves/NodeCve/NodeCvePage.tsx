import React, { useEffect, useState } from 'react';
import { PageSection, Breadcrumb, Divider, BreadcrumbItem, Skeleton } from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';

import { getOverviewPagePath } from '../../utils/searchUtils';
import CvePageHeader, { CveMetadata } from '../../components/CvePageHeader';

const workloadCveOverviewCvePath = getOverviewPagePath('Node', {
    entityTab: 'CVE',
});

function NodeCvePage() {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { cveId } = useParams() as { cveId: string };

    const [nodeCveMetadata, setNodeCveMetadata] = useState<CveMetadata>();
    const nodeCveName = nodeCveMetadata?.cve;

    // TODO - Simulate a loading state, will replace metadata with results from a query
    useEffect(() => {
        setTimeout(() => {
            setNodeCveMetadata({
                cve: cveId,
                firstDiscoveredInSystem: '2021-01-01T00:00:00Z',
                distroTuples: [
                    {
                        summary: 'This is a sample description used during development',
                        link: `https://access.redhat.com/security/cve/${cveId}`,
                        operatingSystem: 'rhel',
                    },
                ],
            });
        }, 1500);
    }, [cveId]);

    return (
        <>
            <PageTitle title={`Node CVEs - NodeCVE ${nodeCveName}`} />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={workloadCveOverviewCvePath}>CVEs</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {nodeCveName ?? (
                            <Skeleton screenreaderText="Loading CVE name" width="200px" />
                        )}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <CvePageHeader data={nodeCveMetadata} />
            </PageSection>
            <Divider component="div" />
            <PageSection className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1">
                <div className="pf-v5-u-background-color-100 pf-v5-u-flex-grow-1"></div>
            </PageSection>
        </>
    );
}

export default NodeCvePage;
