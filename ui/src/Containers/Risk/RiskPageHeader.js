import React, { useEffect } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as deploymentsActions } from 'reducers/deployments';

import { fetchDeployments, fetchDeploymentsCount } from 'services/DeploymentsService';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';

import { pageSize } from 'Components/Table';

function RiskPageHeader({
    currentPage,
    setCurrentDeployments,
    setDeploymentsCount,
    setSelectedDeploymentId,
    isViewFiltered,
    setIsViewFiltered,
    sortOption,
    searchOptions,
    searchModifiers,
    searchSuggestions,
    setSearchOptions,
    setSearchModifiers,
    setSearchSuggestions,
    currentDeployments,
    selectedDeploymentId
}) {
    const hasExecutableFilter =
        searchOptions.length && !searchOptions[searchOptions.length - 1].type;
    const hasNoFilter = !searchOptions.length;

    if (hasExecutableFilter && !isViewFiltered) {
        setIsViewFiltered(true);
    } else if (hasNoFilter && isViewFiltered) {
        setIsViewFiltered(false);
    }
    if (
        hasExecutableFilter &&
        selectedDeploymentId &&
        !currentDeployments.find(deployment => deployment.deployment.id === selectedDeploymentId)
    ) {
        setSelectedDeploymentId(null);
    }

    useEffect(
        () => {
            if (!searchOptions.length || !searchOptions[searchOptions.length - 1].type) {
                fetchDeployments(searchOptions, sortOption, currentPage, pageSize).then(
                    setCurrentDeployments
                );
                fetchDeploymentsCount(searchOptions).then(setDeploymentsCount);
            }
        },
        [searchOptions, sortOption, currentPage, setCurrentDeployments, setDeploymentsCount]
    );

    const subHeader = isViewFiltered ? 'Filtered view' : 'Default view';
    const defaultOption = searchModifiers.find(x => x.value === 'Deployment:');
    return (
        <PageHeader header="Risk" subHeader={subHeader}>
            <SearchInput
                className="w-full"
                id="deployments"
                searchOptions={searchOptions}
                searchModifiers={searchModifiers}
                searchSuggestions={searchSuggestions}
                setSearchOptions={setSearchOptions}
                setSearchModifiers={setSearchModifiers}
                setSearchSuggestions={setSearchSuggestions}
                defaultOption={defaultOption}
                autoCompleteCategories={['DEPLOYMENTS']}
            />
        </PageHeader>
    );
}

RiskPageHeader.propTypes = {
    currentPage: PropTypes.number.isRequired,
    setCurrentDeployments: PropTypes.func.isRequired,
    setDeploymentsCount: PropTypes.func.isRequired,
    setSelectedDeploymentId: PropTypes.func.isRequired,
    isViewFiltered: PropTypes.bool.isRequired,
    setIsViewFiltered: PropTypes.func.isRequired,
    sortOption: PropTypes.shape({}).isRequired,
    currentDeployments: PropTypes.arrayOf(PropTypes.object),
    selectedDeploymentId: PropTypes.string,
    searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
    searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
    searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
    setSearchOptions: PropTypes.func.isRequired,
    setSearchModifiers: PropTypes.func.isRequired,
    setSearchSuggestions: PropTypes.func.isRequired
};

RiskPageHeader.defaultProps = {
    currentDeployments: [],
    selectedDeploymentId: null
};

const mapStateToProps = createStructuredSelector({
    searchOptions: selectors.getDeploymentsSearchOptions,
    searchModifiers: selectors.getDeploymentsSearchModifiers,
    searchSuggestions: selectors.getDeploymentsSearchSuggestions
});

const mapDispatchToProps = {
    setSearchOptions: deploymentsActions.setDeploymentsSearchOptions,
    setSearchModifiers: deploymentsActions.setDeploymentsSearchModifiers,
    setSearchSuggestions: deploymentsActions.setDeploymentsSearchSuggestions
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(RiskPageHeader);
