import React, { useRef } from 'react';
import { useParams } from 'react-router-dom';
import { gql, useQuery } from '@apollo/client';
import {
    PageSection,
    Breadcrumb,
    Divider,
    BreadcrumbItem,
    Skeleton,
    Bullseye,
    Tab,
    TabContent,
    TabTitleText,
    Tabs,
    TabsComponent,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { detailsTabValues } from '../../types';
import { getOverviewPagePath } from '../../utils/searchUtils';

import NodePageHeader, { NodeMetadata, nodeMetadataFragment } from './NodePageHeader';
import NodePageVulnerabilities from './NodePageVulnerabilities';
import NodePageDetails from './NodePageDetails';

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

// TODO - Update for PF5
function NodePage() {
    const { nodeId } = useParams() as { nodeId: string };

    const { data, error } = useQuery<{ node: NodeMetadata }, { id: string }>(nodeMetadataQuery, {
        variables: { id: nodeId },
    });

    const [activeTabKey, setActiveTabKey] = useURLStringUnion('detailsTab', detailsTabValues);

    const vulnTabKey = 'Vulnerabilities';
    const detailsTabKey = 'Details';
    const vulnTabId = 'node-vulnerabilities-tab';
    const detailsTabId = 'node-details-tab';
    const vulnTabRef = useRef<HTMLElement>(null);
    const detailsTabRef = useRef<HTMLElement>(null);

    const nodeName = data?.node?.name ?? '-';

    return (
        <>
            <PageTitle title={`Node CVEs - Node ${nodeName}`} />
            <PageSection variant="light" className="pf-v5-u-py-md">
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
                <PageSection variant="light">
                    <Bullseye>
                        <EmptyStateTemplate
                            title={getAxiosErrorMessage(error)}
                            headingLevel="h2"
                            icon={ExclamationCircleIcon}
                            iconClassName="pf-v5-u-danger-color-100"
                        />
                    </Bullseye>
                </PageSection>
            ) : (
                <>
                    <PageSection variant="light">
                        <NodePageHeader data={data?.node} />
                    </PageSection>
                    <PageSection padding={{ default: 'noPadding' }}>
                        <Tabs
                            activeKey={activeTabKey}
                            onSelect={(e, key) => {
                                setActiveTabKey(key);
                                // pagination.setPage(1);
                            }}
                            component={TabsComponent.nav}
                            className="pf-v5-u-pl-md pf-v5-u-background-color-100"
                            role="region"
                            mountOnEnter
                            unmountOnExit
                        >
                            <Tab
                                eventKey={vulnTabKey}
                                title={<TabTitleText>Vulnerabilities</TabTitleText>}
                                tabContentId={vulnTabId}
                                tabContentRef={vulnTabRef}
                            />
                            <Tab
                                eventKey={detailsTabKey}
                                title={<TabTitleText>Details</TabTitleText>}
                                tabContentId={detailsTabId}
                                tabContentRef={detailsTabRef}
                            />
                        </Tabs>
                    </PageSection>
                    {activeTabKey === vulnTabKey && (
                        <TabContent
                            id={vulnTabId}
                            ref={vulnTabRef}
                            eventKey={vulnTabKey}
                            className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                        >
                            <NodePageVulnerabilities />
                        </TabContent>
                    )}
                    {activeTabKey === detailsTabKey && (
                        <TabContent
                            id={detailsTabId}
                            ref={detailsTabRef}
                            eventKey={detailsTabKey}
                            className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                        >
                            <NodePageDetails />
                        </TabContent>
                    )}
                </>
            )}
        </>
    );
}

export default NodePage;
