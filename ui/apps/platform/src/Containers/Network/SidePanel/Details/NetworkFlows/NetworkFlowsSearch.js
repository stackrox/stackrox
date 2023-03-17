import React from 'react';
import PropTypes from 'prop-types';

import useAutoCompleteResults from 'hooks/useAutoCompleteResults';
import SearchInput, { createSearchModifiers } from 'Components/SearchInput';
import getNetworkFlowValueByCategory from './networkFlowUtils/getNetworkFlowValueByCategory';

const networkFlowCategories = [
    'Traffic',
    'Entity',
    'Type',
    'Namespace',
    'Protocols',
    'Ports',
    'Connection',
];
const networkFlowSearchModifiers = createSearchModifiers(networkFlowCategories);

const NetworkFlowsSearch = ({ networkFlows, searchOptions, setSearchOptions }) => {
    const autoCompleteResults = useAutoCompleteResults(
        networkFlows,
        searchOptions,
        networkFlowCategories,
        getNetworkFlowValueByCategory
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
};

NetworkFlowsSearch.propTypes = {
    networkFlows: PropTypes.arrayOf(
        PropTypes.shape({
            traffic: PropTypes.string.isRequired,
            deploymentId: PropTypes.string.isRequired,
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
    searchOptions: PropTypes.arrayOf(PropTypes.shape),
    setSearchOptions: PropTypes.func.isRequired,
};

NetworkFlowsSearch.defaultProps = {
    networkFlows: [],
    searchOptions: [],
};

export default NetworkFlowsSearch;
