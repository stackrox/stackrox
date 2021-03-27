import React from 'react';
import PropTypes from 'prop-types';

import useAutoCompleteResults from 'hooks/useAutoCompleteResults';
import { networkFlowStatus } from 'constants/networkGraph';

import SearchInput, { createSearchModifiers } from 'Components/SearchInput';

export const searchCategories = [
    'Status',
    'Entity',
    'Traffic',
    'Type',
    'Namespace',
    'Port',
    'Protocol',
    'State',
];

const dataResolversByCategory = {
    Status: (datum) => datum.status,
    Entity: (datum) => datum.peer.entity.name,
    Traffic: (datum) => {
        if (datum.peer.ingress && datum.peer.egress) {
            return 'bidirectional';
        }
        if (datum.peer.ingress) {
            return 'ingress';
        }
        return 'egress';
    },
    Type: (datum) => datum.peer.entity.type,
    Namespace: (datum) => datum.peer.entity.namespace,
    Port: (datum) => datum.peer.port,
    Protocol: (datum) => datum.peer.protocol,
    State: (datum) => datum.peer.state,
};

export function getNetworkBaselineValueByCategory(datum, category) {
    return dataResolversByCategory[category]?.(datum);
}

const networkFlowSearchModifiers = createSearchModifiers(searchCategories);

function NetworkBaselinesSearch({ networkBaselines, searchOptions, setSearchOptions }) {
    const autoCompleteResults = useAutoCompleteResults(
        networkBaselines,
        searchOptions,
        searchCategories,
        getNetworkBaselineValueByCategory
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

NetworkBaselinesSearch.propTypes = {
    networkBaselines: PropTypes.arrayOf(
        PropTypes.shape({
            peer: PropTypes.shape({
                entity: PropTypes.shape({
                    id: PropTypes.string.isRequired,
                    type: PropTypes.string.isRequired,
                    name: PropTypes.bool,
                    namespace: PropTypes.string,
                }),
                port: PropTypes.string.isRequired,
                protocol: PropTypes.string.isRequired,
                ingress: PropTypes.bool.isRequired,
                state: PropTypes.string.isRequired,
            }),
            status: PropTypes.oneOf(Object.values(networkFlowStatus)).isRequired,
        })
    ),
    searchOptions: PropTypes.arrayOf(PropTypes.shape),
    setSearchOptions: PropTypes.func.isRequired,
};

NetworkBaselinesSearch.defaultProps = {
    networkBaselines: [],
    searchOptions: [],
};

export default NetworkBaselinesSearch;
