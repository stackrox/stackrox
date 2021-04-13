import React from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { actions as searchActions } from 'reducers/network/search';
import { SearchEntry, SearchState } from 'reducers/pageSearch';
import {
    ORCHESTRATOR_COMPONENT_KEY,
    orchestratorComponentOption,
} from 'Containers/Navigation/OrchestratorComponentsToggle';
import ReduxSearchInput from 'Containers/Search/ReduxSearchInput';

function NetworkSearch({
    searchOptions,
    searchModifiers,
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

    let prependAutocompleteQuery;
    const orchestratorComponentShowState = localStorage.getItem(ORCHESTRATOR_COMPONENT_KEY);
    if (orchestratorComponentShowState !== 'true') {
        prependAutocompleteQuery = [...orchestratorComponentOption];
    }

    return (
        <ReduxSearchInput
            className="w-full pl-2"
            searchOptions={searchOptions}
            searchModifiers={searchModifiers}
            setSearchOptions={setSearchOptions}
            setSearchSuggestions={setSearchSuggestions}
            onSearch={onSearch}
            isDisabled={isDisabled}
            prependAutocompleteQuery={prependAutocompleteQuery}
        />
    );
}

const mapStateToProps = createStructuredSelector<
    SearchState,
    {
        searchOptions: SearchEntry[];
        searchModifiers: SearchEntry[];
    }
>({
    searchOptions: selectors.getNetworkSearchOptions,
    searchModifiers: selectors.getNetworkSearchModifiers,
});

const mapDispatchToProps = {
    setSearchOptions: searchActions.setNetworkSearchOptions,
    setSearchSuggestions: searchActions.setNetworkSearchSuggestions,
    closeSidePanel: pageActions.closeSidePanel,
};

export default connect(mapStateToProps, mapDispatchToProps)(NetworkSearch);
