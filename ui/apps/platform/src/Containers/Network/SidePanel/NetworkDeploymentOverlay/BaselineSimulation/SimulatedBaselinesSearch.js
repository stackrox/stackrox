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
        const { simulatedStatus } = datum;
        // In order to grab the values from the right sub-object, we can use the simulated status as a key
        const statusKey = simulatedStatus.toLowerCase();
        const { ingress, egress } =
            simulatedStatus === 'MODIFIED' ? datum.peer[statusKey].added : datum.peer[statusKey];
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
        const { simulatedStatus } = datum;
        // In order to grab the values from the right sub-object, we can use the simulated status as a key
        const statusKey = simulatedStatus.toLowerCase();
        const { port } =
            simulatedStatus === 'MODIFIED' ? datum.peer[statusKey].added : datum.peer[statusKey];
        return port;
    },
    Protocol: (datum) => {
        const { simulatedStatus } = datum;
        // In order to grab the values from the right sub-object, we can use the simulated status as a key
        const statusKey = simulatedStatus.toLowerCase();
        const { protocol } =
            simulatedStatus === 'MODIFIED' ? datum.peer[statusKey].added : datum.peer[statusKey];
        return protocol;
    },
    State: (datum) => datum.peer.state,
};

export function getSimulatedBaselineValueByCategory(datum, category) {
    return dataResolversByCategory[category]?.(datum);
}

const networkFlowSearchModifiers = createSearchModifiers(searchCategories);

function SimulatedBaselinesSearch({ networkBaselines, searchOptions, setSearchOptions }) {
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
            placeholder="Add one or more resource filters"
        />
    );
}

export default SimulatedBaselinesSearch;
