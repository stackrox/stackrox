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
import { DEFAULT_VM_PAGE_SIZE } from 'Containers/Vulnerabilities/constants';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { getVirtualMachine } from 'services/VirtualMachineService';

import { detailsTabValues } from '../../types';
import { getOverviewPagePath } from '../../utils/searchUtils';
import {
    COMPONENT_SORT_FIELD,
    CVE_EPSS_PROBABILITY_SORT_FIELD,
    CVE_SEVERITY_SORT_FIELD,
    CVE_SORT_FIELD,
    CVSS_SORT_FIELD,
} from '../../utils/sortFields';
import VirtualMachinePageHeader from './VirtualMachinePageHeader';
import VirtualMachinePagePackages from './VirtualMachinePagePackages';
import VirtualMachinePageVulnerabilities from './VirtualMachinePageVulnerabilities';

const VULNERABILITIES_TAB_ID = 'vulnerabilities-tab-content';
const PACKAGES_TAB_ID = 'packages-tab-content';

const virtualMachineCveOverviewPath = getOverviewPagePath('VirtualMachine', {
    entityTab: 'VirtualMachine',
});

const sortFields = [
    COMPONENT_SORT_FIELD,
    CVE_EPSS_PROBABILITY_SORT_FIELD,
    CVE_SORT_FIELD,
    CVE_SEVERITY_SORT_FIELD,
    CVSS_SORT_FIELD,
];

const defaultPackagesSortOption = { field: COMPONENT_SORT_FIELD, direction: 'asc' } as const;

const defaultVulnerabilitiesSortOption = {
    field: CVE_SEVERITY_SORT_FIELD,
    direction: 'desc',
} as const;

function VirtualMachinePage() {
    const { virtualMachineId } = useParams() as { virtualMachineId: string };
    const urlPagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const urlSearch = useURLSearch();
    const urlSorting = useURLSort({
        sortFields,
        defaultSortOption: defaultVulnerabilitiesSortOption,
        onSort: () => urlPagination.setPage(1, 'replace'),
    });

    const fetchVirtualMachine = useCallback(
        () => getVirtualMachine(virtualMachineId),
        [virtualMachineId]
    );

    const { data: virtualMachineData, isLoading, error } = useRestQuery(fetchVirtualMachine);

    const [activeTabKey, setActiveTabKey] = useURLStringUnion('detailsTab', detailsTabValues);

    const vulnTabKey = detailsTabValues[0];
    const packagesTabKey = detailsTabValues[4];

    const virtualMachineName = virtualMachineData?.name;

    function onTabChange(value: string | number) {
        if (value === packagesTabKey) {
            urlSorting.setSortOption(defaultPackagesSortOption);
        } else {
            urlSorting.setSortOption(defaultVulnerabilitiesSortOption);
        }
        setActiveTabKey(value);
        urlPagination.setPage(1, 'replace');
        urlSearch.setSearchFilter({});
    }

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
                        onTabChange(key);
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
                            urlSearch={urlSearch}
                            urlSorting={urlSorting}
                            urlPagination={urlPagination}
                        />
                    </TabContent>
                )}
                {activeTabKey === packagesTabKey && (
                    <TabContent id={PACKAGES_TAB_ID}>
                        <VirtualMachinePagePackages
                            virtualMachineData={virtualMachineData}
                            isLoadingVirtualMachineData={isLoading}
                            errorVirtualMachineData={error}
                            urlSearch={urlSearch}
                            urlSorting={urlSorting}
                            urlPagination={urlPagination}
                        />
                    </TabContent>
                )}
            </PageSection>
        </>
    );
}

export default VirtualMachinePage;
