import React from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Title,
} from '@patternfly/react-core';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';

function NamespaceViewPage() {
    return (
        <>
            <PageTitle title="Workload CVEs - Namespace view" />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={vulnerabilitiesWorkloadCvesPath}>
                        Workload CVEs
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Namespace view</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <Flex
                    direction={{ default: 'column' }}
                    alignItems={{ default: 'alignItemsFlexStart' }}
                >
                    <Title headingLevel="h1" className="pf-u-mb-sm">
                        Namespace view
                    </Title>
                    <FlexItem>Discover and prioritize namespaces by risk priority</FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection padding={{ default: 'noPadding' }}></PageSection>
        </>
    );
}

export default NamespaceViewPage;
