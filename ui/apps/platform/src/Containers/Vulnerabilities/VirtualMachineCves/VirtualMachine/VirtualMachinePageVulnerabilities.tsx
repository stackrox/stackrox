import React from 'react';
import { PageSection } from '@patternfly/react-core';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied } from 'utils/searchUtils';
import AdvancedFiltersToolbar from 'Containers/Vulnerabilities/components/AdvancedFiltersToolbar';
import {
    virtualMachineCVESearchFilterConfig,
    namespaceSearchFilterConfig,
    clusterSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';

export type VirtualMachinePageVulnerabilitiesProps = {
    virtualMachineId: string;
};

const searchFilterConfig = [
    virtualMachineCVESearchFilterConfig,
    namespaceSearchFilterConfig,
    clusterSearchFilterConfig,
];

function VirtualMachinePageVulnerabilities({
    virtualMachineId,
}: VirtualMachinePageVulnerabilitiesProps) {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const isFiltered = getHasSearchApplied(searchFilter);

    return (
        <PageSection
            padding={{ default: 'noPadding' }}
            isFilled
            className="pf-v5-u-display-flex pf-v5-u-flex-direction-column"
        >
            <AdvancedFiltersToolbar
                className="pf-v5-u-px-sm pf-v5-u-pb-0"
                searchFilter={searchFilter}
                searchFilterConfig={searchFilterConfig}
                onFilterChange={(newFilter) => {
                    setSearchFilter(newFilter);
                }}
            />
            <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100 pf-v5-u-p-lg">
                <EmptyStateTemplate title="Virtual Machine Vulnerabilities" headingLevel="h2">
                    Virtual machine vulnerabilities table will be implemented here for{' '}
                    {virtualMachineId}.{isFiltered && ' Filters are applied.'}
                </EmptyStateTemplate>
            </div>
        </PageSection>
    );
}

export default VirtualMachinePageVulnerabilities;
