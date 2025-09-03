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
    name: `rhel-vm${id}`,
    namespace: 'prod-cluster/rhacs',
    description: 'RHEL virtual machine for production workloads',
    status: 'Running',
    ipAddress: '10.10.10.10',
    operatingSystem: 'RHEL',
    guestOS: 'RHEL 9.4',
    agent: 'Falcon',
    scanTime: '2025-08-04T16:45:10Z',
    createdAt: '2025-07-21T09:12:00Z',
    owner: 'No owner',
    pod: 'Not available',
    template: 'rhel9-template',
    bootOrder: ['disk-0 (Disk)'],
    workloadProfile: 'Desktop',
    cdroms: [
        {
            name: 'cdrom0',
            source: 'containerdisk://rhel-9.4.iso',
        },
    ],
    labels: [
        { key: 'environment', value: 'production' },
        { key: 'tier', value: 'frontend' },
    ],
    annotations: [
        { key: 'kubevirt.io/latest-observed-api-version', value: 'v1' },
        { key: 'vm.kubevirt.io/os', value: 'rhel9.4' },
    ],
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
