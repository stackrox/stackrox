import { useCallback } from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import {
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    PageSection,
    Skeleton,
    Tab,
    TabContent,
    Tabs,
    Text,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { getVirtualMachine } from 'services/VirtualMachineService';

import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
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
import VirtualMachinePageComponents from './VirtualMachinePageComponents';
import VirtualMachinePageVulnerabilities from './VirtualMachinePageVulnerabilities';

const VULNERABILITIES_TAB_ID = 'vulnerabilities-tab-content';
const COMPONENTS_TAB_ID = 'components-tab-content';
const DETAILS_TAB_ID = 'details-tab-content';

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

const defaultComponentsSortOption = { field: COMPONENT_SORT_FIELD, direction: 'asc' } as const;

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
    const componentsTabKey = detailsTabValues[4];
    const detailsTabKey = detailsTabValues[1];

    const virtualMachineName = virtualMachineData?.name;

    function onTabChange(value: string | number) {
        if (value === componentsTabKey) {
            urlSorting.setSortOption(defaultComponentsSortOption);
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
                        eventKey={componentsTabKey}
                        tabContentId={COMPONENTS_TAB_ID}
                        title={componentsTabKey}
                    />
                    <Tab
                        eventKey={detailsTabKey}
                        tabContentId={DETAILS_TAB_ID}
                        title={detailsTabKey}
                    />
                </Tabs>
            </PageSection>
            <PageSection variant="light" padding={{ default: 'padding' }}>
                <Text>
                    <Text>
                        {activeTabKey === vulnTabKey &&
                            'Prioritize and remediate observed CVEs for this virtual machine'}
                        {activeTabKey === componentsTabKey &&
                            'View all components from this virtual machine'}
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
                {activeTabKey === componentsTabKey && (
                    <TabContent id={COMPONENTS_TAB_ID}>
                        <VirtualMachinePageComponents
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
