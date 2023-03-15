import React from 'react';

import useAutoCompleteResults from 'hooks/useAutoCompleteResults';
import SearchInput, { createSearchModifiers } from 'Components/SearchInput';

export const searchCategories = [
    'Entity',
    'Traffic',
    'Type',
    'Namespace',
    'Port',
    'Protocol',
    'State',
];

const dataResolversByCategory = {
    Entity: (datum) => datum.peer.entity.name,
    Traffic: (datum) => {
        const { ingress, egress } = datum.peer;
        if (ingress && egress) {
            return 'bidirectional';
        }
        if (ingress) {
            return 'ingress';
        }
        return 'egress';
    },
    Type: (datum) => datum.peer.entity.type,
    Namespace: (datum) => datum.peer.entity.namespace,
    Port: (datum) => {
        return datum.peer.port;
    },
    Protocol: (datum) => {
        return datum.peer.protocol;
    },
    State: (datum) => datum.peer.state,
};

export function getSimulatedBaselineValueByCategory(datum, category) {
    return dataResolversByCategory[category]?.(datum);
}

const networkFlowSearchModifiers = createSearchModifiers(searchCategories);

function BaselineSimulationSearch({ networkBaselines, searchOptions, setSearchOptions }) {
    const autoCompleteResults = useAutoCompleteResults(
        networkBaselines,
        searchOptions,
        searchCategories,
        getSimulatedBaselineValueByCategory
    );

    return (
        <SearchInput
            className="w-full"
            searchOptions={searchOptions}
            searchModifiers={networkFlowSearchModifiers}
            setSearchOptions={setSearchOptions}
            autoCompleteResults={autoCompleteResults}
            placeholder="Filter deployments"
        />
    );
}

export default BaselineSimulationSearch;
