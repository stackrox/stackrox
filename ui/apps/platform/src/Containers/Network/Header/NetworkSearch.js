import React, { useEffect, useState } from 'react';
import { connect } from 'react-redux';
import { useHistory } from 'react-router-dom';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { actions as searchActions } from 'reducers/network/search';
import useURLSearch from 'hooks/useURLSearch';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { getSearchOptionsForCategory } from 'services/SearchService';
import { isCompleteSearchFilter } from 'utils/searchUtils';
import SearchFilterInput from 'Components/SearchFilterInput';
import {
    ORCHESTRATOR_COMPONENTS_KEY,
    orchestratorComponentsOption,
} from 'utils/orchestratorComponents';

import './NetworkSearch.css';

function searchFilterToSearchEntries(searchFilter) {
    const entries = [];
    Object.entries(searchFilter).forEach(([key, value]) => {
        entries.push({ label: `${key}:`, value: `${key}:`, type: 'categoryOption' });
        if (value !== '') {
            const values = Array.isArray(value) ? value : [value];
            const valueOptions = values.map((v) => ({ label: v, value: v }));
            entries.push(...valueOptions);
        }
    });
    return entries;
}

const searchCategory = 'DEPLOYMENTS';
const searchOptionExclusions = ['Cluster', 'Namespace', 'Namespace ID', 'Orchestrator Component'];

function NetworkSearch({
    selectedNamespaceFilters,
    dispatchSearchFilter,
    closeSidePanel,
    isDisabled,
}) {
    const history = useHistory();
    const [searchOptions, setSearchOptions] = useState([]);
    const { searchFilter, setSearchFilter } = useURLSearch();

    useEffect(() => {
        const { request, cancel } = getSearchOptionsForCategory(searchCategory);
        request
            .then((options) => {
                const filteredOptions = options.filter((o) => !searchOptionExclusions.includes(o));
                setSearchOptions(filteredOptions);
            })
            .catch(() => {
                // A request error will disable the search filter.
            });

        return cancel;
    }, [setSearchOptions]);

    // Keep the Redux store in sync with the URL Search Filter
    useEffect(() => {
        dispatchSearchFilter(searchFilterToSearchEntries(searchFilter));
    }, [searchFilter, dispatchSearchFilter]);

    function onSearch(options) {
        setSearchFilter(options);
        if (isCompleteSearchFilter(options)) {
            history.push(`/main/network${history.location.search}`);
            closeSidePanel();
        }
    }

    const orchestratorComponentShowState = localStorage.getItem(ORCHESTRATOR_COMPONENTS_KEY);
    const prependAutocompleteQuery =
        orchestratorComponentShowState !== 'true' ? [...orchestratorComponentsOption] : [];

    if (selectedNamespaceFilters.length) {
        prependAutocompleteQuery.push({ value: 'Namespace:', type: 'categoryOption' });
        selectedNamespaceFilters.forEach((nsFilter) =>
            prependAutocompleteQuery.push({ value: nsFilter })
        );
    }

    return (
        <SearchFilterInput
            className="pf-u-w-100 pf-search-shim"
            placeholder="Add one or more deployment filters"
            searchFilter={searchFilter}
            searchCategory="DEPLOYMENTS"
            searchOptions={searchOptions}
            handleChangeSearchFilter={onSearch}
            autocompleteQueryPrefix={searchOptionsToQuery(prependAutocompleteQuery)}
            isDisabled={isDisabled}
        />
    );
}

const mapStateToProps = createStructuredSelector({
    selectedNamespaceFilters: selectors.getSelectedNamespaceFilters,
});

const mapDispatchToProps = {
    dispatchSearchFilter: searchActions.setNetworkSearchOptions,
    closeSidePanel: pageActions.closeSidePanel,
};

export default connect(mapStateToProps, mapDispatchToProps)(NetworkSearch);
