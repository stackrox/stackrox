import React from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { actions as searchActions } from 'reducers/network/search';
import {
    ORCHESTRATOR_COMPONENT_KEY,
    orchestratorComponentOption,
} from 'Containers/Navigation/OrchestratorComponentsToggle';
import ReduxSearchInput from 'Containers/Search/ReduxSearchInput';

import './NetworkSearch.css';

function NetworkSearch({
    searchOptions,
    searchModifiers,
    selectedNamespaceFilters,
    setSearchOptions,
    setSearchSuggestions,
    closeSidePanel,
    isDisabled,
}) {
    function onSearch(options) {
        if (options.length && !options[options.length - 1].type) {
            closeSidePanel();
        }
    }

    const orchestratorComponentShowState = localStorage.getItem(ORCHESTRATOR_COMPONENT_KEY);
    const prependAutocompleteQuery =
        orchestratorComponentShowState !== 'true' ? [...orchestratorComponentOption] : [];

    if (selectedNamespaceFilters.length) {
        prependAutocompleteQuery.push({ value: 'Namespace:', type: 'categoryOption' });
        selectedNamespaceFilters.forEach((nsFilter) =>
            prependAutocompleteQuery.push({ value: nsFilter })
        );
    }

    return (
        <ReduxSearchInput
            className="pf-u-w-100 network-search"
            placeholder="Add one or more deployment filters"
            searchOptions={searchOptions}
            searchModifiers={searchModifiers}
            setSearchOptions={setSearchOptions}
            setSearchSuggestions={setSearchSuggestions}
            onSearch={onSearch}
            isDisabled={isDisabled}
            prependAutocompleteQuery={prependAutocompleteQuery}
            autoCompleteCategories={['DEPLOYMENTS']}
        />
    );
}

const mapStateToProps = createStructuredSelector({
    searchOptions: selectors.getNetworkSearchOptions,
    searchModifiers: selectors.getNetworkSearchModifiers,
    selectedNamespaceFilters: selectors.getSelectedNamespaceFilters,
});

const mapDispatchToProps = {
    setSearchOptions: searchActions.setNetworkSearchOptions,
    setSearchSuggestions: searchActions.setNetworkSearchSuggestions,
    closeSidePanel: pageActions.closeSidePanel,
};

export default connect(mapStateToProps, mapDispatchToProps)(NetworkSearch);
