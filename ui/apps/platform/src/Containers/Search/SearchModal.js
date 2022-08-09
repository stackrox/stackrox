import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { selectors } from 'reducers';

import PageHeader from 'Components/PageHeader';
import ReduxSearchInput from 'Containers/Search/ReduxSearchInput';
import SearchResults from 'Containers/Search/SearchResults';
import * as Icon from 'react-feather';
import onClickOutside from 'react-onclickoutside';

class SearchModal extends Component {
    static propTypes = {
        searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setSearchOptions: PropTypes.func.isRequired,
        setSearchModifiers: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,
        onClose: PropTypes.func.isRequired,
    };

    componentDidMount() {
        document.addEventListener('keydown', this.handleKeyDown);
    }

    componentWillUnmount() {
        document.removeEventListener('keydown', this.handleKeyDown);
    }

    handleKeyDown = (event) => {
        // 'escape' key maps to keycode '27'
        if (event.keyCode === 27) {
            this.props.onClose();
        }
    };

    handleClickOutside = () => {
        this.props.onClose();
    };

    render() {
        return (
            <div className="flex flex-col h-full w-full">
                <div className="flex w-full z-xs-101">
                    <PageHeader header="Search All:">
                        <ReduxSearchInput
                            className="w-full flex-shrink"
                            searchOptions={this.props.searchOptions}
                            searchModifiers={this.props.searchModifiers}
                            searchSuggestions={this.props.searchSuggestions}
                            setSearchOptions={this.props.setSearchOptions}
                            setSearchModifiers={this.props.setSearchModifiers}
                            setSearchSuggestions={this.props.setSearchSuggestions}
                            isGlobal
                        />
                        <div className="flex items-center border-l border-base-300 h-full w-20 -mr-4 ml-4">
                            <button
                                type="button"
                                className="w-full h-full px-4 hover:bg-base-200 flex items-center justify-center"
                                onClick={this.props.onClose}
                            >
                                <Icon.X size="28" />
                            </button>
                        </div>
                    </PageHeader>
                </div>
                <SearchResults onClose={this.props.onClose} />
            </div>
        );
    }
}

const SearchModalContainer = (props) => {
    const EnhancedSearchModal = onClickOutside(SearchModal);
    return (
        <div className="search-modal w-full z-md-300 pf-u-background-color-100">
            <EnhancedSearchModal {...props} />
        </div>
    );
};

const mapStateToProps = createStructuredSelector({
    searchOptions: selectors.getGlobalSearchOptions,
    searchModifiers: selectors.getGlobalSearchModifiers,
    searchSuggestions: selectors.getGlobalSearchSuggestions,
});

const mapDispatchToProps = (dispatch) => ({
    setSearchOptions: (searchOptions) =>
        dispatch(globalSearchActions.setGlobalSearchOptions(searchOptions)),
    setSearchModifiers: (searchModifiers) =>
        dispatch(globalSearchActions.setGlobalSearchModifiers(searchModifiers)),
    setSearchSuggestions: (searchSuggestions) =>
        dispatch(globalSearchActions.setGlobalSearchSuggestions(searchSuggestions)),
});

export default connect(mapStateToProps, mapDispatchToProps)(SearchModalContainer);
