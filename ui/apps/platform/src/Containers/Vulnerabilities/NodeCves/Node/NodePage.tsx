import { useParams } from 'react-router-dom-v5-compat';
import { gql, useQuery } from '@apollo/client';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    PageSection,
    Skeleton,
    Tab,
    Tabs,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { detailsTabValues } from '../../types';
import { getOverviewPagePath } from '../../utils/searchUtils';

import NodePageHeader, { nodeMetadataFragment } from './NodePageHeader';
import type { NodeMetadata } from './NodePageHeader';
import NodePageVulnerabilities from './NodePageVulnerabilities';
import NodePageDetails from './NodePageDetails';

const idDetails = 'NodePageDetails';
const idVulnerabilities = 'NodePageVulnerabilities';

const nodeCveOverviewPath = getOverviewPagePath('Node', {
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

function NodePage() {
    const { nodeId } = useParams() as { nodeId: string };

    const { data, error } = useQuery<{ node: NodeMetadata }, { id: string }>(nodeMetadataQuery, {
        variables: { id: nodeId },
    });

    const [activeTabKey, setActiveTabKey] = useURLStringUnion('detailsTab', detailsTabValues);

    const vulnTabKey = detailsTabValues[0];
    const detailTabKey = detailsTabValues[1];

    const nodeName = data?.node?.name ?? '-';

    return (
        <>
            <PageTitle title={`Node CVEs - Node ${nodeName}`} />
            <PageSection type="breadcrumb">
                <Breadcrumb>
                    <BreadcrumbItemLink to={nodeCveOverviewPath}>Nodes</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {nodeName ?? (
                            <Skeleton screenreaderText="Loading Node name" width="200px" />
                        )}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            {error ? (
                <PageSection>
                    <Bullseye>
                        <EmptyStateTemplate
                            title={getAxiosErrorMessage(error)}
                            headingLevel="h2"
                            icon={ExclamationCircleIcon}
                            status="danger"
                        />
                    </Bullseye>
                </PageSection>
            ) : (
                <>
                    <PageSection>
                        <NodePageHeader data={data?.node} />
                    </PageSection>
                    <PageSection type="tabs">
                        <Tabs
                            activeKey={activeTabKey}
                            onSelect={(e, key) => {
                                setActiveTabKey(key);
                                // pagination.setPage(1);
                            }}
                            usePageInsets
                            mountOnEnter
                            unmountOnExit
                        >
                            <Tab
                                eventKey={vulnTabKey}
                                tabContentId={idVulnerabilities}
                                title={vulnTabKey}
                            >
                                <NodePageVulnerabilities nodeId={nodeId} />
                            </Tab>
                            <Tab
                                eventKey={detailTabKey}
                                tabContentId={idDetails}
                                title={detailTabKey}
                            >
                                <NodePageDetails nodeId={nodeId} />
                            </Tab>
                        </Tabs>
                    </PageSection>
                </>
            )}
        </>
    );
}

export default NodePage;
