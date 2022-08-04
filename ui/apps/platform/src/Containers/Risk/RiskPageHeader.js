import React from 'react';
import PropTypes from 'prop-types';

import entityTypes, { searchCategories } from 'constants/entityTypes';
import PageHeader from 'Components/PageHeader';
import {
    ORCHESTRATOR_COMPONENTS_KEY,
    orchestratorComponentsOption,
} from 'utils/orchestratorComponents';
import SearchFilterInput from 'Components/SearchFilterInput';
import useURLSearch from 'hooks/useURLSearch';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import CreatePolicyFromSearch from './CreatePolicyFromSearch';

function RiskPageHeader({ isViewFiltered, searchOptions }) {
    const { searchFilter, setSearchFilter } = useURLSearch();
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
                placeholder="Add one or more resource filters"
                handleChangeSearchFilter={(filter) => setSearchFilter(filter, 'push')}
                autocompleteQueryPrefix={searchOptionsToQuery(prependAutocompleteQuery)}
            />
            <CreatePolicyFromSearch />
        </PageHeader>
    );
}

RiskPageHeader.propTypes = {
    isViewFiltered: PropTypes.bool.isRequired,
    searchOptions: PropTypes.arrayOf(PropTypes.string).isRequired,
};

export default RiskPageHeader;
