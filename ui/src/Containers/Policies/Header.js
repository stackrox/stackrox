import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as searchActions } from 'reducers/policies/search';
import { actions as tableActions } from 'reducers/policies/table';
import { createSelector, createStructuredSelector } from 'reselect';

import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';

// Header Is the top line with the search box.
class Header extends Component {
    static propTypes = {
        selectPolicyIds: PropTypes.func.isRequired,

        history: ReactRouterPropTypes.history.isRequired,

        searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setSearchOptions: PropTypes.func.isRequired,
        setSearchModifiers: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,

        isViewFiltered: PropTypes.bool.isRequired
    };

    onSearch = searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            // reset table selection on search.
            this.props.selectPolicyIds([]);
            this.props.history.push('/main/policies');
        }
    };

    render() {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        const defaultOption = this.props.searchModifiers.find(x => x.value === 'Policy:');
        return (
            <PageHeader header="Policies" subHeader={subHeader}>
                <SearchInput
                    className="w-full"
                    id="policies"
                    searchOptions={this.props.searchOptions}
                    searchModifiers={this.props.searchModifiers}
                    searchSuggestions={this.props.searchSuggestions}
                    setSearchOptions={this.props.setSearchOptions}
                    setSearchModifiers={this.props.setSearchModifiers}
                    setSearchSuggestions={this.props.setSearchSuggestions}
                    onSearch={this.onSearch}
                    defaultOption={defaultOption}
                    autoCompleteCategories={['POLICIES']}
                />
            </PageHeader>
        );
    }
}

const isViewFiltered = createSelector(
    [selectors.getPoliciesSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    searchOptions: selectors.getPoliciesSearchOptions,
    searchModifiers: selectors.getPoliciesSearchModifiers,
    searchSuggestions: selectors.getPoliciesSearchSuggestions,
    isViewFiltered
});

const mapDispatchToProps = {
    selectPolicyIds: tableActions.selectPolicyIds,
    setSearchOptions: searchActions.setPoliciesSearchOptions,
    setSearchModifiers: searchActions.setPoliciesSearchModifiers,
    setSearchSuggestions: searchActions.setPoliciesSearchSuggestions
};

export default withRouter(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(Header)
);
