import type { ReactElement, ReactNode } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Content,
    Flex,
    FlexItem,
    PageSection,
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
        <PageSection hasBodyWrapper={false} component="div">
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
                        <Content component="p">
                            Cluster registration secrets contain secrets for secured cluster
                            services to establish initial trust with Central.
                        </Content>
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
