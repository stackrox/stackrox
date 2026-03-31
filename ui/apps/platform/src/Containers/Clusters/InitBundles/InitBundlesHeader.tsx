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
import { clustersBasePath, clustersInitBundlesPath } from 'routePaths';

export const titleInitBundles = 'Cluster init bundles';

export type InitBundlesHeaderProps = {
    headerActions?: ReactNode | null;
    title: string;
};

function InitBundlesHeader({ headerActions, title }: InitBundlesHeaderProps): ReactElement {
    return (
        <>
            <PageTitle title={title} />
            <PageSection type="breadcrumb">
                <Breadcrumb>
                    <BreadcrumbItemLink to={clustersBasePath}>Clusters</BreadcrumbItemLink>
                    {title !== titleInitBundles && (
                        <BreadcrumbItemLink to={clustersInitBundlesPath}>
                            {titleInitBundles}
                        </BreadcrumbItemLink>
                    )}
                    <BreadcrumbItem isActive>{title}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <PageSection component="div">
                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        <Title headingLevel="h1">{title}</Title>
                        <Content component="p">
                            Cluster init bundles contain secrets for secured cluster services to
                            authenticate with Central.
                        </Content>
                    </Flex>
                    {headerActions && (
                        <FlexItem align={{ default: 'alignRight' }}>{headerActions}</FlexItem>
                    )}
                </Flex>
            </PageSection>
        </>
    );
}

export default InitBundlesHeader;
