import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';

import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import {
    virtualMachineCVESearchFilterConfig,
    virtualMachineComponentSearchFilterConfig,
    virtualMachinesClusterSearchFilterConfig,
    virtualMachinesNamespaceSearchFilterConfig,
    virtualMachinesSearchFilterConfig,
} from '../../searchFilterConfig';

const searchFilterConfig = [
    virtualMachinesClusterSearchFilterConfig,
    virtualMachineCVESearchFilterConfig,
    virtualMachinesNamespaceSearchFilterConfig,
    virtualMachinesSearchFilterConfig,
    virtualMachineComponentSearchFilterConfig,
];

// Shared filter toolbar for both overview tables, which each render their own toolbar.
// Factored out to make sure the two tables stay in sync. State is read from the URL, not props.
function VirtualMachineCvesFilterToolbar() {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { setPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);

    return (
        <AdvancedFiltersToolbar
            searchFilter={searchFilter}
            searchFilterConfig={searchFilterConfig}
            defaultSearchFilterEntity="Virtual machine"
            includeCveSeverityFilters={false}
            includeCveStatusFilters={false}
            onFilterChange={(newFilter) => {
                setSearchFilter(newFilter);
                setPage(1, 'replace');
            }}
        />
    );
}

export default VirtualMachineCvesFilterToolbar;
