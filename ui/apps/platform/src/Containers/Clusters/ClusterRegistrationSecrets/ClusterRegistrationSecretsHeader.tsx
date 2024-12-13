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
import { clustersBasePath, clustersClusterRegistrationSecretsPath } from 'routePaths';

export const titleClusterRegistrationSecrets = 'Cluster registration secrets';

export type ClusterRegistrationSecretsHeaderProps = {
    headerActions?: ReactNode | null;
    title: string;
};

function ClusterRegistrationSecretsHeader({
    headerActions,
    title,
}: ClusterRegistrationSecretsHeaderProps): ReactElement {
    return (
        <PageSection component="div" variant="light">
            <PageTitle title={title} />
            <Flex direction={{ default: 'column' }}>
                <Breadcrumb>
                    <BreadcrumbItemLink to={clustersBasePath}>Clusters</BreadcrumbItemLink>
                    {title !== titleClusterRegistrationSecrets && (
                        <BreadcrumbItemLink to={clustersClusterRegistrationSecretsPath}>
                            {titleClusterRegistrationSecrets}
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
                            Cluster registration secrets contain secrets for secured cluster
                            services to establish initial trust with Central.
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

export default ClusterRegistrationSecretsHeader;
