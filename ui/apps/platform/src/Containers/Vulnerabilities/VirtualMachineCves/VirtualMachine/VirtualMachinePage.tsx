import React from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import {
    PageSection,
    Breadcrumb,
    Divider,
    BreadcrumbItem,
    Skeleton,
    Tab,
    TabContent,
    Tabs,
    Text,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useURLStringUnion from 'hooks/useURLStringUnion';

import { detailsTabValues } from '../../types';
import { getOverviewPagePath } from '../../utils/searchUtils';

import VirtualMachinePageHeader, { VirtualMachineMetadata } from './VirtualMachinePageHeader';
import VirtualMachinePageVulnerabilities from './VirtualMachinePageVulnerabilities';
import VirtualMachinePageDetails from './VirtualMachinePageDetails';

const VULNERABILITIES_TAB_ID = 'vulnerabilities-tab-content';
const DETAILS_TAB_ID = 'details-tab-content';

const virtualMachineCveOverviewPath = getOverviewPagePath('VirtualMachine', {
    entityTab: 'VirtualMachine',
});

// Mock data for virtual machine
const getMockVirtualMachineData = (id: string): VirtualMachineMetadata => ({
    id,
    name: id,
    guestOS: 'RHEL 9.4',
    location: 'prod-cluster/rhacs',
    agent: '007',
    scanTime: '2025-01-05T02:12:12Z',
    created: '2024-02-20T10:00:42Z',
});

function VirtualMachinePage() {
    const { virtualMachineId } = useParams() as { virtualMachineId: string };

    const virtualMachineData = getMockVirtualMachineData(virtualMachineId);

    const [activeTabKey, setActiveTabKey] = useURLStringUnion('detailsTab', detailsTabValues);

    const vulnTabKey = detailsTabValues[0];
    const detailTabKey = detailsTabValues[1];

    const virtualMachineName = virtualMachineData?.name ?? '-';

    return (
        <>
            <PageTitle title={`Virtual Machine CVEs - Virtual Machine ${virtualMachineName}`} />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={virtualMachineCveOverviewPath}>
                        Virtual Machines
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {virtualMachineName ?? (
                            <Skeleton
                                screenreaderText="Loading Virtual Machine name"
                                width="200px"
                            />
                        )}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <VirtualMachinePageHeader data={virtualMachineData} />
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                <Tabs
                    activeKey={activeTabKey}
                    onSelect={(_, key) => {
                        setActiveTabKey(key);
                    }}
                    className="pf-v5-u-pl-md pf-v5-u-background-color-100"
                >
                    <Tab
                        eventKey={vulnTabKey}
                        tabContentId={VULNERABILITIES_TAB_ID}
                        title={vulnTabKey}
                    />
                    <Tab
                        eventKey={detailTabKey}
                        tabContentId={DETAILS_TAB_ID}
                        title={detailTabKey}
                    />
                </Tabs>
            </PageSection>
            <PageSection variant="light" padding={{ default: 'padding' }}>
                <Text>
                    {activeTabKey === vulnTabKey
                        ? 'Prioritize and remediate observed CVEs for this virtual machine'
                        : 'View details about this virtual machine'}
                </Text>
            </PageSection>
            <PageSection
                isFilled
                padding={{ default: 'padding' }}
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column"
                aria-label={activeTabKey}
                role="tabpanel"
                tabIndex={0}
            >
                {activeTabKey === vulnTabKey && (
                    <TabContent id={VULNERABILITIES_TAB_ID}>
                        <VirtualMachinePageVulnerabilities virtualMachineId={virtualMachineId} />
                    </TabContent>
                )}
                {activeTabKey === detailTabKey && (
                    <TabContent id={DETAILS_TAB_ID}>
                        <VirtualMachinePageDetails virtualMachineId={virtualMachineId} />
                    </TabContent>
                )}
            </PageSection>
        </>
    );
}

export default VirtualMachinePage;
