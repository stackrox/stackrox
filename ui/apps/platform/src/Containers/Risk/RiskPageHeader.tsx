import entityTypes, { searchCategories } from 'constants/entityTypes';
import PageHeader from 'Components/PageHeader';
import {
    ORCHESTRATOR_COMPONENTS_KEY,
    orchestratorComponentsOption,
} from 'utils/orchestratorComponents';
import SearchFilterInput from 'Components/SearchFilterInput';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import usePermissions from 'hooks/usePermissions';
import searchOptionsToQuery from 'services/searchOptionsToQuery';

import CreatePolicyFromSearch from './CreatePolicyFromSearch';
import type { SearchFilter } from 'types/search';

type RiskPageHeaderProps = {
    isViewFiltered: boolean;
    searchOptions: string[];
    searchFilter: SearchFilter;
    onSearch: (newSearchFilter: SearchFilter) => void;
};

function RiskPageHeader({
    isViewFiltered,
    searchOptions,
    searchFilter,
    onSearch,
}: RiskPageHeaderProps) {
    const isRouteEnabled = useIsRouteEnabled();
    const { hasReadWriteAccess } = usePermissions();
    // Require READ_WRITE_ACCESS to create plus READ_ACCESS to other resources for Policies route.
    const hasWriteAccessForCreatePolicy =
        hasReadWriteAccess('WorkflowAdministration') && isRouteEnabled('policy-management');

    const subHeader = isViewFiltered ? 'Filtered view' : 'Default view';
    const autoCompleteCategory = searchCategories[entityTypes.DEPLOYMENT];

    const orchestratorComponentShowState = localStorage.getItem(ORCHESTRATOR_COMPONENTS_KEY);
    const prependAutocompleteQuery =
        orchestratorComponentShowState !== 'true' ? orchestratorComponentsOption : [];
    return (
        <PageHeader header="Risk" subHeader={subHeader}>
            <SearchFilterInput
                className="w-full"
                searchFilter={searchFilter}
                searchOptions={searchOptions}
                searchCategory={autoCompleteCategory}
                placeholder="Filter deployments"
                handleChangeSearchFilter={onSearch}
                autocompleteQueryPrefix={searchOptionsToQuery(prependAutocompleteQuery)}
            />
            {hasWriteAccessForCreatePolicy && <CreatePolicyFromSearch />}
        </PageHeader>
    );
}

export default RiskPageHeader;
