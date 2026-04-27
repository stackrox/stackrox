import { useEffect, useState } from 'react';

import SearchFilterInput from 'Components/SearchFilterInput';
import { getSearchOptionsForCategory } from 'services/SearchService';

import { useSearchFilter } from '../NetworkGraphURLStateContext';

import './NetworkSearch.css';

const searchCategory = 'DEPLOYMENTS';
const searchOptionExclusions = ['Cluster', 'Deployment', 'Namespace', 'Namespace ID'];

type NetworkSearchProps = {
    selectedCluster: string;
    selectedNamespaces: string[];
    selectedDeployments: string[];
    isDisabled: boolean;
};

function NetworkSearch({
    selectedCluster,
    selectedNamespaces,
    selectedDeployments,
    isDisabled,
}: NetworkSearchProps) {
    const [searchOptions, setSearchOptions] = useState<string[]>([]);
    const { searchFilter, setSearchFilter } = useSearchFilter();

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

    return (
        <SearchFilterInput
            className="pf-v6-u-w-100 pf-search-shim"
            placeholder="Filter deployments"
            searchFilter={searchFilter}
            searchCategory="DEPLOYMENTS"
            searchOptions={searchOptions}
            handleChangeSearchFilter={onSearch}
            isDisabled={isDisabled}
        />
    );
}

export default NetworkSearch;
