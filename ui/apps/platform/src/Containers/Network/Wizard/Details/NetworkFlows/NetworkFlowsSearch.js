import React, { useState, useMemo } from 'react';
import PropTypes from 'prop-types';

import SearchInput, { getLastModifier, createSearchModifiers } from 'Components/SearchInput';

const networkFlowCategories = [
    'Traffic',
    'Deployment',
    'Namespace',
    'Protocols',
    'Ports',
    'Connection',
];
const networkFlowSearchModifiers = createSearchModifiers(networkFlowCategories);

const categoryToFieldResolverMap = {
    Traffic: (datum) => datum.traffic,
    Deployment: (datum) => datum.deploymentName,
    Namespace: (datum) => datum.namespace,
    Protocols: (datum) => datum.portsAndProtocols.map((d) => d.protocol),
    Ports: (datum) => datum.portsAndProtocols.map((d) => d.port),
    Connection: (datum) => datum.connection,
};

export function getValueByCategory(datum, category) {
    const resolveData = categoryToFieldResolverMap[category];
    const value = resolveData && resolveData(datum);
    return value;
}

export function getNetworkFlowsAutoCompleteResultsMap({ networkFlows }) {
    const autoCompleteResultsMap = networkFlows.reduce((acc, networkFlow) => {
        networkFlowCategories.forEach((category) => {
            const value = getValueByCategory(networkFlow, category);
            if (!acc[category]) {
                acc[category] = new Set();
            }
            // if the value is an array, we need to add each item to the set
            if (Array.isArray(value)) {
                value.forEach((datum) => acc[category].add(datum.toString()));
            } else {
                acc[category].add(value.toString());
            }
        });
        return acc;
    }, {});
    // convert the Set -> Array for each category
    Object.keys(autoCompleteResultsMap).forEach((category) => {
        autoCompleteResultsMap[category] = Array.from(autoCompleteResultsMap[category]);
    });
    return autoCompleteResultsMap;
}

const NetworkFlowsSearch = ({ networkFlows }) => {
    const [searchOptions, setSearchOptions] = useState([]);

    const autoCompleteResultsMap = useMemo(
        () =>
            getNetworkFlowsAutoCompleteResultsMap({
                networkFlows,
            }),
        [networkFlows]
    );
    const category = getLastModifier(searchOptions);
    const autoCompleteResults = autoCompleteResultsMap[category] || [];

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
};

NetworkFlowsSearch.propTypes = {
    networkFlows: PropTypes.arrayOf(
        PropTypes.shape({
            traffic: PropTypes.string.isRequired,
            deploymentName: PropTypes.string.isRequired,
            namespace: PropTypes.string.isRequired,
            portsAndProtocols: PropTypes.arrayOf(
                PropTypes.shape({
                    port: PropTypes.number.isRequired,
                    protocol: PropTypes.string.isRequired,
                })
            ),
            connection: PropTypes.string.isRequired,
        })
    ),
};

NetworkFlowsSearch.defaultProps = {
    networkFlows: [],
};

export default NetworkFlowsSearch;
