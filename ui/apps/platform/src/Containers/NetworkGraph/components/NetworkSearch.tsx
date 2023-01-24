import React, { useEffect, useState } from 'react';

import SearchFilterInput from 'Components/SearchFilterInput';
import useURLSearch from 'hooks/useURLSearch';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { getSearchOptionsForCategory } from 'services/SearchService';
import { orchestratorComponentsOption } from 'utils/orchestratorComponents';

import './NetworkSearch.css';

const searchCategory = 'DEPLOYMENTS';
const searchOptionExclusions = [
    'Cluster',
    'Deployment',
    'Namespace',
    'Namespace ID',
    'Orchestrator Component',
];

type NetworkSearchsProps = {
    selectedCluster?: string;
    selectedNamespaces?: string[];
    selectedDeployments?: string[];
};

function NetworkSearch({
    selectedCluster = '',
    selectedNamespaces = [],
    selectedDeployments = [],
}: NetworkSearchsProps) {
    const [searchOptions, setSearchOptions] = useState<string[]>([]);
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

    function onSearch(options) {
        const newOptions = { ...options };
        newOptions.Cluster = selectedCluster;
        newOptions.Namespace = selectedNamespaces;
        newOptions.Deployment = selectedDeployments;

        setSearchFilter(newOptions);
    }

    const prependAutocompleteQuery = [...orchestratorComponentsOption];

    return (
        <SearchFilterInput
            className="pf-u-w-100 theme-light pf-search-shim"
            placeholder="Add one or more deployment filters"
            searchFilter={searchFilter}
            searchCategory="DEPLOYMENTS"
            searchOptions={searchOptions}
            autocompleteQueryPrefix={searchOptionsToQuery(prependAutocompleteQuery)}
            handleChangeSearchFilter={onSearch}
        />
    );
}

export default NetworkSearch;
