import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as deploymentsActions, types } from 'reducers/deployments';

import NoResultsMessage from 'Components/NoResultsMessage';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import Table from 'Components/Table';
import Panel from 'Components/Panel';
import Tabs from 'Components/Tabs';
import Loader from 'Components/Loader';
import TabContent from 'Components/TabContent';
import { sortNumber } from 'sorters/sorters';
import RiskDetails from './RiskDetails';
import DeploymentDetails from './DeploymentDetails';

class RiskPage extends Component {
    static propTypes = {
        deployments: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setSearchOptions: PropTypes.func.isRequired,
        setSearchModifiers: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,
        isViewFiltered: PropTypes.bool.isRequired,
        history: ReactRouterPropTypes.history.isRequired,
        location: ReactRouterPropTypes.location.isRequired,
        match: ReactRouterPropTypes.match.isRequired,
        isFetchingDeployment: PropTypes.bool.isRequired
    };

    getSelectedDeployment = () => {
        if (this.props.match.params.id) {
            return this.props.deployments.find(
                deployment => deployment.id === this.props.match.params.id
            );
        }
        return null;
    };

    updateSelectedDeployment = deployment => {
        const urlSuffix = deployment && deployment.id ? `/${deployment.id}` : '';
        this.props.history.push({
            pathname: `/main/risk${urlSuffix}`,
            search: this.props.location.search
        });
    };

    renderTable() {
        const columns = [
            { key: 'name', label: 'Name' },
            { key: 'cluster', label: 'Cluster' },
            { key: 'namespace', label: 'Namespace' },
            { key: 'priority', label: 'Priority', sortMethod: sortNumber('priority') }
        ];
        const rows = this.props.deployments;
        if (!rows.length)
            return <NoResultsMessage message="No results found. Please refine your search." />;
        return <Table columns={columns} rows={rows} onRowClick={this.updateSelectedDeployment} />;
    }

    renderSidePanel = () => {
        const selectedDeployment = this.getSelectedDeployment();
        if (!selectedDeployment) return null;

        const riskPanelTabs = [{ text: 'Risk Indicators' }, { text: 'Deployment Details' }];
        const content = this.props.isFetchingDeployment ? (
            <Loader />
        ) : (
            <Tabs headers={riskPanelTabs}>
                <TabContent>
                    <div className="flex flex-1 flex-col">
                        <RiskDetails risk={selectedDeployment.risk} />
                    </div>
                </TabContent>
                <TabContent>
                    <div className="flex flex-1 flex-col">
                        <DeploymentDetails deployment={selectedDeployment} />
                    </div>
                </TabContent>
            </Tabs>
        );

        return (
            <div className="w-1/2">
                <Panel header={selectedDeployment.name} onClose={this.updateSelectedDeployment}>
                    {content}
                </Panel>
            </div>
        );
    };

    render() {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        return (
            <section className="flex flex-1 h-full">
                <div className="flex flex-1 flex-col">
                    <PageHeader header="Risk" subHeader={subHeader}>
                        <SearchInput
                            id="risk"
                            searchOptions={this.props.searchOptions}
                            searchModifiers={this.props.searchModifiers}
                            searchSuggestions={this.props.searchSuggestions}
                            setSearchOptions={this.props.setSearchOptions}
                            setSearchModifiers={this.props.setSearchModifiers}
                            setSearchSuggestions={this.props.setSearchSuggestions}
                        />
                    </PageHeader>
                    <div className="flex flex-1">
                        <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow bg-base-100 flex flex-1">
                            {this.renderTable()}
                        </div>
                        {this.renderSidePanel()}
                    </div>
                </div>
            </section>
        );
    }
}

const isViewFiltered = createSelector(
    [selectors.getDeploymentsSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    deployments: selectors.getFilteredDeployments,
    searchOptions: selectors.getDeploymentsSearchOptions,
    searchModifiers: selectors.getDeploymentsSearchModifiers,
    searchSuggestions: selectors.getDeploymentsSearchSuggestions,
    isViewFiltered,
    isFetchingDeployment: state => selectors.getLoadingStatus(state, types.FETCH_DEPLOYMENT)
});

const mapDispatchToProps = (dispatch, props) => ({
    setSearchOptions: searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            props.history.push('/main/risk');
        }
        dispatch(deploymentsActions.setDeploymentsSearchOptions(searchOptions));
    },
    setSearchModifiers: deploymentsActions.setDeploymentsSearchModifiers,
    setSearchSuggestions: deploymentsActions.setDeploymentsSearchSuggestions
});

export default connect(mapStateToProps, mapDispatchToProps)(RiskPage);
