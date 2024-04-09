import React from 'react';
import { useParams } from 'react-router-dom';
import { gql, useQuery } from '@apollo/client';
import {
    PageSection,
    Breadcrumb,
    Divider,
    BreadcrumbItem,
    Skeleton,
    Bullseye,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import NodePageHeader, { NodeMetadata, nodeMetadataFragment } from './NodePageHeader';
import { getOverviewPagePath } from '../../utils/searchUtils';

const workloadCveOverviewCvePath = getOverviewPagePath('Node', {
    entityTab: 'Node',
});

const nodeMetadataQuery = gql`
    ${nodeMetadataFragment}
    query getNodeMetadata($id: ID!) {
        node(id: $id) {
            ...NodeMetadata
        }
    }
`;

// TODO - Update for PF5
function NodePage() {
    const { nodeId } = useParams() as { nodeId: string };

    const { data, error } = useQuery<{ node: NodeMetadata }, { id: string }>(nodeMetadataQuery, {
        variables: { id: nodeId },
    });

    const nodeName = data?.node?.name ?? '-';

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
            {error ? (
                <PageSection variant="light">
                    <Bullseye>
                        <EmptyStateTemplate
                            title={getAxiosErrorMessage(error)}
                            headingLevel="h2"
                            icon={ExclamationCircleIcon}
                            iconClassName="pf-u-danger-color-100"
                        />
                    </Bullseye>
                </PageSection>
            ) : (
                <>
                    <PageSection variant="light">
                        <NodePageHeader data={data?.node} />
                    </PageSection>
                </>
            )}
        </>
    );
}

export default NodePage;
