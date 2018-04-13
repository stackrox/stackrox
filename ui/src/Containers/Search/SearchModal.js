import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { selectors } from 'reducers';

import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import SearchResults from 'Containers/Search/SearchResults';
import * as Icon from 'react-feather';

const SearchModal = props => (
    <div className="search-modal pl-4 pr-4 border-t border-base-300 w-full absolute">
        <div className="flex flex-col h-full w-full">
            <div className="flex flex-row w-full bg-white">
                <PageHeader header="Search All:">
                    <SearchInput
                        searchOptions={props.searchOptions}
                        searchModifiers={props.searchModifiers}
                        searchSuggestions={props.searchSuggestions}
                        setSearchOptions={props.setSearchOptions}
                        setSearchModifiers={props.setSearchModifiers}
                        setSearchSuggestions={props.setSearchSuggestions}
                    />
                </PageHeader>
                <button
                    className="flex items-center justify-center border-b border-base-300 border-l px-4 hover:bg-base-200"
                    onClick={props.onClose}
                >
                    <Icon.X className="h-4 w-4" />
                </button>
            </div>
            <SearchResults onClose={props.onClose} />
        </div>
    </div>
);

SearchModal.propTypes = {
    searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
    searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
    searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
    setSearchOptions: PropTypes.func.isRequired,
    setSearchModifiers: PropTypes.func.isRequired,
    setSearchSuggestions: PropTypes.func.isRequired,
    onClose: PropTypes.func.isRequired
};

const mapStateToProps = createStructuredSelector({
    searchOptions: selectors.getGlobalSearchOptions,
    searchModifiers: selectors.getGlobalSearchModifiers,
    searchSuggestions: selectors.getGlobalSearchSuggestions
});

const mapDispatchToProps = dispatch => ({
    setSearchOptions: searchOptions =>
        dispatch(globalSearchActions.setGlobalSearchOptions(searchOptions)),
    setSearchModifiers: searchModifiers =>
        dispatch(globalSearchActions.setGlobalSearchModifiers(searchModifiers)),
    setSearchSuggestions: searchSuggestions =>
        dispatch(globalSearchActions.setGlobalSearchSuggestions(searchSuggestions))
});

export default connect(mapStateToProps, mapDispatchToProps)(SearchModal);
