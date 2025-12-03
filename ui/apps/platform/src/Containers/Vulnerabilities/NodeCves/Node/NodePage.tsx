import { useParams } from 'react-router-dom-v5-compat';
import { gql, useQuery } from '@apollo/client';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Divider,
    PageSection,
    Skeleton,
    Tab,
    TabContent,
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
            <PageSection hasBodyWrapper={false} className="pf-v6-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={nodeCveOverviewPath}>Nodes</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {nodeName ?? (
                            <Skeleton screenreaderText="Loading Node name" width="200px" />
                        )}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            {error ? (
                <PageSection hasBodyWrapper={false}>
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
                    <PageSection hasBodyWrapper={false}>
                        <NodePageHeader data={data?.node} />
                    </PageSection>
                    <PageSection hasBodyWrapper={false} padding={{ default: 'noPadding' }}>
                        <Tabs
                            activeKey={activeTabKey}
                            onSelect={(e, key) => {
                                setActiveTabKey(key);
                                // pagination.setPage(1);
                            }}
                            className="pf-v6-u-pl-md pf-v6-u-background-color-100"
                        >
                            <Tab
                                eventKey={vulnTabKey}
                                tabContentId={idVulnerabilities}
                                title={vulnTabKey}
                            />
                            <Tab
                                eventKey={detailTabKey}
                                tabContentId={idDetails}
                                title={detailTabKey}
                            />
                        </Tabs>
                    </PageSection>
                    <PageSection
                        hasBodyWrapper={false}
                        isFilled
                        padding={{ default: 'noPadding' }}
                        className="pf-v6-u-display-flex pf-v6-u-flex-direction-column"
                        aria-label={activeTabKey}
                        role="tabpanel"
                        tabIndex={0}
                    >
                        {activeTabKey === vulnTabKey && (
                            <TabContent id={idVulnerabilities}>
                                <NodePageVulnerabilities nodeId={nodeId} />
                            </TabContent>
                        )}
                        {activeTabKey === detailTabKey && (
                            <TabContent id={idDetails}>
                                <NodePageDetails nodeId={nodeId} />
                            </TabContent>
                        )}
                    </PageSection>
                </>
            )}
        </>
    );
}

export default NodePage;
