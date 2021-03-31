import React from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/network/page';
import { actions as searchActions } from 'reducers/network/search';
import { SearchEntry, SearchState } from 'reducers/pageSearch';

import ReduxSearchInput from 'Containers/Search/ReduxSearchInput';

function NetworkSearch({
    searchOptions,
    searchModifiers,
    setSearchOptions,
    setSearchSuggestions,
    closeWizard,
    isDisabled,
}) {
    function onSearch(options) {
        if (options.length && !options[options.length - 1].type) {
            closeWizard();
        }
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
            prependQuery
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
    closeWizard: pageActions.closeNetworkWizard,
};

export default connect(mapStateToProps, mapDispatchToProps)(NetworkSearch);
