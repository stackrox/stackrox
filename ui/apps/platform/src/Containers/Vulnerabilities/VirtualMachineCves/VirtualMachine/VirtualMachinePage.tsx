import React, { useCallback } from 'react';
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
import EmptyStateTemplate from 'Components/EmptyStateTemplate/EmptyStateTemplate';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useRestQuery from 'hooks/useRestQuery';

import { getVirtualMachine, type VirtualMachine } from 'services/VirtualMachineService';
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

// Maps API response to header component format
function mapVirtualMachineToMetadata(vm: VirtualMachine): VirtualMachineMetadata {
    return {
        id: vm.id,
        name: vm.name,
        namespace: `${vm.clusterName}/${vm.namespace}`,
        description: vm.facts?.description || 'Virtual machine',
        status: vm.facts?.status || 'Unknown',
        ipAddress: vm.facts?.ipAddress || 'Unknown',
        operatingSystem: vm.facts?.operatingSystem || 'Unknown',
        guestOS: vm.facts?.guestOS || 'Unknown',
        agent: vm.facts?.agent || 'Unknown',
        scanTime: vm.scan?.scanTime,
        createdAt: vm.lastUpdated,
        owner: vm.facts?.owner || 'No owner',
        pod: vm.facts?.pod || 'Not available',
        template: vm.facts?.template || 'Unknown',
        bootOrder: vm.facts?.bootOrder ? vm.facts.bootOrder.split(',') : [],
        workloadProfile: vm.facts?.workloadProfile || 'Unknown',
        cdroms: vm.facts?.cdroms ? JSON.parse(vm.facts.cdroms) : [],
        labels: Object.entries(vm.facts || {})
            .filter(([key]) => key.startsWith('label:'))
            .map(([key, value]) => ({ key: key.replace('label:', ''), value })),
        annotations: Object.entries(vm.facts || {})
            .filter(([key]) => key.startsWith('annotation:'))
            .map(([key, value]) => ({ key: key.replace('annotation:', ''), value })),
    };
}

function VirtualMachinePage() {
    const { virtualMachineId } = useParams() as { virtualMachineId: string };

    const fetchVirtualMachine = useCallback(
        () => getVirtualMachine(virtualMachineId),
        [virtualMachineId]
    );

    const { data: virtualMachine, isLoading, error } = useRestQuery(fetchVirtualMachine);

    const virtualMachineData = virtualMachine
        ? mapVirtualMachineToMetadata(virtualMachine)
        : undefined;

    const [activeTabKey, setActiveTabKey] = useURLStringUnion('detailsTab', detailsTabValues);

    const vulnTabKey = detailsTabValues[0];
    const detailTabKey = detailsTabValues[1];

    const virtualMachineName = virtualMachineData?.name;

    return (
        <>
            <PageTitle
                title={`Virtual Machine CVEs - Virtual Machine ${virtualMachineName || (isLoading ? 'Loading...' : 'Error')}`}
            />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={virtualMachineCveOverviewPath}>
                        Virtual Machines
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {error
                            ? 'Error'
                            : (virtualMachineName ?? (
                                  <Skeleton
                                      screenreaderText="Loading Virtual Machine name"
                                      width="200px"
                                  />
                              ))}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            {!error && (
                <PageSection variant="light">
                    <VirtualMachinePageHeader
                        data={virtualMachineData}
                        isLoading={isLoading}
                        error={error}
                    />
                </PageSection>
            )}
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
                {error ? (
                    <EmptyStateTemplate title="Unable to load virtual machine" headingLevel="h2">
                        {error.message}
                    </EmptyStateTemplate>
                ) : isLoading ? (
                    <Skeleton width="100%" height="200px" />
                ) : (
                    <>
                        {activeTabKey === vulnTabKey && (
                            <TabContent id={VULNERABILITIES_TAB_ID}>
                                <VirtualMachinePageVulnerabilities
                                    virtualMachineId={virtualMachineId}
                                />
                            </TabContent>
                        )}
                        {activeTabKey === detailTabKey && (
                            <TabContent id={DETAILS_TAB_ID}>
                                <VirtualMachinePageDetails virtualMachineId={virtualMachineId} />
                            </TabContent>
                        )}
                    </>
                )}
            </PageSection>
        </>
    );
}

export default VirtualMachinePage;
