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
import useRestQuery from 'hooks/useRestQuery';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { getVirtualMachine } from 'services/VirtualMachineService';

import { detailsTabValues } from '../../types';
import { getOverviewPagePath } from '../../utils/searchUtils';
import VirtualMachinePageHeader from './VirtualMachinePageHeader';
import VirtualMachinePageVulnerabilities from './VirtualMachinePageVulnerabilities';

const VULNERABILITIES_TAB_ID = 'vulnerabilities-tab-content';
const PACKAGES_TAB_ID = 'packages-tab-content';

const virtualMachineCveOverviewPath = getOverviewPagePath('VirtualMachine', {
    entityTab: 'VirtualMachine',
});

function VirtualMachinePage() {
    const { virtualMachineId } = useParams() as { virtualMachineId: string };

    const fetchVirtualMachine = useCallback(
        () => getVirtualMachine(virtualMachineId),
        [virtualMachineId]
    );

    const { data: virtualMachineData, isLoading, error } = useRestQuery(fetchVirtualMachine);

    const [activeTabKey, setActiveTabKey] = useURLStringUnion('detailsTab', detailsTabValues);

    const vulnTabKey = detailsTabValues[0];
    const packagesTabKey = detailsTabValues[4];

    const virtualMachineName = virtualMachineData?.name;

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
                <VirtualMachinePageHeader
                    virtualMachineData={virtualMachineData}
                    isLoading={isLoading}
                    error={error}
                />
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
                        eventKey={packagesTabKey}
                        tabContentId={PACKAGES_TAB_ID}
                        title={packagesTabKey}
                    />
                </Tabs>
            </PageSection>
            <PageSection variant="light" padding={{ default: 'padding' }}>
                <Text>
                    <Text>
                        {activeTabKey === vulnTabKey &&
                            'Prioritize and remediate observed CVEs for this virtual machine'}
                        {activeTabKey === packagesTabKey &&
                            'View all packages from this virtual machine'}
                    </Text>
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
                        <VirtualMachinePageVulnerabilities
                            virtualMachineData={virtualMachineData}
                            isLoadingVirtualMachineData={isLoading}
                            errorVirtualMachineData={error}
                        />
                    </TabContent>
                )}
                {activeTabKey === packagesTabKey && (
                    <TabContent id={PACKAGES_TAB_ID}>packages table here</TabContent>
                )}
            </PageSection>
        </>
    );
}

export default VirtualMachinePage;
