import React, { ReactElement, ReactNode } from 'react';
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

export const titleInitBundles = 'Cluster init bundles';

export type InitBundlesHeaderProps = {
    headerActions?: ReactNode | null;
    title: string;
};

function InitBundlesHeader({ headerActions, title }: InitBundlesHeaderProps): ReactElement {
    return (
        <PageSection component="div" variant="light">
            <PageTitle title={title} />
            <Flex direction={{ default: 'column' }}>
                <Breadcrumb>
                    <BreadcrumbItemLink to={clustersBasePath}>Clusters</BreadcrumbItemLink>
                    {title !== titleInitBundles && (
                        <BreadcrumbItemLink to={clustersInitBundlesPath}>
                            {titleInitBundles}
                        </BreadcrumbItemLink>
                    )}
                    <BreadcrumbItem isActive>{title}</BreadcrumbItem>
                </Breadcrumb>
                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        <Title headingLevel="h1">{title}</Title>
                        <Text>
                            Cluster init bundles contain secrets for secured cluster services to
                            authenticate with Central.
                        </Text>
                    </Flex>
                    {headerActions && (
                        <FlexItem align={{ default: 'alignRight' }}>{headerActions}</FlexItem>
                    )}
                </Flex>
            </Flex>
        </PageSection>
    );
}

export default InitBundlesHeader;
