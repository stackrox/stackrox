import React, { useEffect, useState } from 'react';
import { useHistory } from 'react-router-dom';

import useURLSearch from 'hooks/useURLSearch';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { getSearchOptionsForCategory } from 'services/SearchService';
import { isCompleteSearchFilter } from 'utils/searchUtils';
import { SearchEntry, SearchFilter } from 'types/search';
import SearchFilterInput from 'Components/SearchFilterInput';

import './NetworkSearch.css';

function searchFilterToSearchEntries(searchFilter): SearchEntry[] {
    const entries: SearchEntry[] = [];
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
const searchOptionExclusions = [
    'Cluster',
    'Deployment',
    'Namespace',
    'Namespace ID',
    'Orchestrator Component',
];

function NetworkSearch() {
    const history = useHistory();
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
        setSearchFilter(options);
        if (isCompleteSearchFilter(options)) {
            history.push(`/main/network-graph${history.location.search as string}`);
        }
    }

    return (
        <SearchFilterInput
            className="pf-u-w-100 pf-search-shim"
            placeholder="Add one or more deployment filters"
            searchFilter={searchFilter}
            searchCategory="DEPLOYMENTS"
            searchOptions={searchOptions}
            handleChangeSearchFilter={onSearch}
        />
    );
}

export default NetworkSearch;
