import React from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { actions as searchAutoCompleteActions } from 'reducers/searchAutocomplete';
import { selectors } from 'reducers';

import SearchInput, { searchInputPropTypes, searchInputDefaultProps } from 'Components/SearchInput';

const ReduxSearchInput = ({
    className,
    placeholder,
    searchOptions,
    searchModifiers,
    setSearchOptions,
    setSearchSuggestions,
    onSearch,
    isGlobal,
    defaultOption,
    autoCompleteResults,
    sendAutoCompleteRequest,
    clearAutoComplete,
    autoCompleteCategories,
    setAllSearchOptions,
    isDisabled,
    prependAutocompleteQuery,
}) => {
    return (
        <SearchInput
            className={className}
            placeholder={placeholder}
            searchOptions={searchOptions}
            searchModifiers={searchModifiers}
            setSearchOptions={setSearchOptions}
            setSearchSuggestions={setSearchSuggestions}
            onSearch={onSearch}
            isGlobal={isGlobal}
            defaultOption={defaultOption}
            autoCompleteResults={autoCompleteResults}
            sendAutoCompleteRequest={sendAutoCompleteRequest}
            clearAutoComplete={clearAutoComplete}
            autoCompleteCategories={autoCompleteCategories}
            setAllSearchOptions={setAllSearchOptions}
            isDisabled={isDisabled}
            prependAutocompleteQuery={prependAutocompleteQuery}
        />
    );
};

ReduxSearchInput.propTypes = searchInputPropTypes;
ReduxSearchInput.defaultProps = searchInputDefaultProps;

const mapStateToProps = createStructuredSelector({
    autoCompleteResults: selectors.getAutoCompleteResults,
});

const mapDispatchToProps = {
    sendAutoCompleteRequest: searchAutoCompleteActions.sendAutoCompleteRequest,
    clearAutoComplete: searchAutoCompleteActions.clearAutoComplete,
    setAllSearchOptions: searchAutoCompleteActions.setAllSearchOptions,
};

/** @deprecated */
export default connect(mapStateToProps, mapDispatchToProps)(ReduxSearchInput);
