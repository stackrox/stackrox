import React, { ReactElement } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Flex,
    FlexItem,
    PageSection,
    Text,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import { clustersBasePath, clustersInitBundlesPath } from 'routePaths';

const titleInitBundles = 'Cluster init bundles';

export type InitBundlesHeaderProps = {
    alignRightElement?: ReactElement | null;
    titleNotInitBundles?: string;
};

function InitBundlesHeader({
    alignRightElement,
    titleNotInitBundles,
}: InitBundlesHeaderProps): ReactElement {
    const title = titleNotInitBundles ?? titleInitBundles;
    return (
        <PageSection component="div" variant="light">
            <PageTitle title={title} />
            <Flex direction={{ default: 'column' }}>
                <Breadcrumb>
                    <BreadcrumbItemLink to={clustersBasePath}>Clusters</BreadcrumbItemLink>
                    {titleNotInitBundles && (
                        <BreadcrumbItemLink to={clustersInitBundlesPath}>
                            {titleInitBundles}
                        </BreadcrumbItemLink>
                    )}
                    <BreadcrumbItem isActive>{title}</BreadcrumbItem>
                </Breadcrumb>
                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">{title}</Title>
                        <Text>
                            Cluster init bundles contain secrets for secured cluster services to
                            authenticate with Central.
                        </Text>
                    </FlexItem>
                    {alignRightElement && (
                        <FlexItem align={{ default: 'alignRight' }}>{alignRightElement}</FlexItem>
                    )}
                </Flex>
            </Flex>
        </PageSection>
    );
}

export default InitBundlesHeader;
