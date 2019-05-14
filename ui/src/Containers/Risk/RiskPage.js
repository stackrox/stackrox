import React, { useState } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as deploymentsActions } from 'reducers/deployments';

import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import Panel from 'Components/Panel';
import Tabs from 'Components/Tabs';
import Loader from 'Components/Loader';
import TabContent from 'Components/TabContent';
import TablePagination from 'Components/TablePagination';

import RiskDetails from './RiskDetails';
import DeploymentDetails from './DeploymentDetails';
import ProcessDetails from './ProcessDetails';
import RiskTable from './RiskTable';

const RiskPage = ({
    history,
    location,
    deployments,
    selectedDeployment,
    searchOptions,
    processGroup,
    isViewFiltered,
    searchModifiers,
    searchSuggestions,
    setSearchOptions,
    setSearchModifiers,
    setSearchSuggestions
}) => {
    const [page, setPage] = useState(0);

    function onSearch(options) {
        if (options.length && !options[options.length - 1].type) {
            history.push('/main/risk');
        }
    }

    function updateSelectedDeployment(deployment) {
        const urlSuffix = deployment && deployment.id ? `/${deployment.id}` : '';
        history.push({
            pathname: `/main/risk${urlSuffix}`,
            search: location.search
        });
    }

    function renderPanel() {
        const { length } = deployments;
        const paginationComponent = (
            <TablePagination page={page} dataLength={length} setPage={setPage} />
        );
        const isFiltering = searchOptions.length;
        const headerText = `${length} Deployment${length === 1 ? '' : 's'} ${
            isFiltering ? 'Matched' : ''
        }`;
        return (
            <Panel header={headerText} headerComponents={paginationComponent}>
                <div className="w-full">
                    <RiskTable
                        rows={deployments}
                        selectedDeployment={selectedDeployment}
                        processGroup={processGroup}
                        page={page}
                    />
                </div>
            </Panel>
        );
    }

    function renderSidePanel() {
        if (!selectedDeployment) return null;

        const riskPanelTabs = [{ text: 'Risk Indicators' }, { text: 'Deployment Details' }];
        if (processGroup.groups !== undefined && processGroup.groups.length !== 0) {
            riskPanelTabs.push({ text: 'Process Discovery' });
        }

        const content =
            selectedDeployment && selectedDeployment.risk === undefined ? (
                <Loader />
            ) : (
                <Tabs headers={riskPanelTabs}>
                    <TabContent>
                        <div className="flex flex-1 flex-col pb-5">
                            <RiskDetails risk={selectedDeployment.risk} />
                        </div>
                    </TabContent>
                    <TabContent>
                        <div className="flex flex-1 flex-col relative">
                            <div className="absolute w-full">
                                <DeploymentDetails deployment={selectedDeployment} />
                            </div>
                        </div>
                    </TabContent>
                    <TabContent>
                        <div className="flex flex-1 flex-col relative">
                            <ProcessDetails processGroup={processGroup} />
                        </div>
                    </TabContent>
                </Tabs>
            );

        return (
            <Panel
                header={selectedDeployment.name}
                className="bg-primary-200 z-10 w-full h-full absolute pin-r pin-t md:w-1/2 min-w-72 md:relative"
                onClose={updateSelectedDeployment}
            >
                {content}
            </Panel>
        );
    }

    const subHeader = isViewFiltered ? 'Filtered view' : 'Default view';
    const defaultOption = searchModifiers.find(x => x.value === 'Deployment:');
    return (
        <section className="flex flex-1 flex-col h-full">
            <div className="flex flex-1 flex-col">
                <PageHeader header="Risk" subHeader={subHeader}>
                    <SearchInput
                        className="w-full"
                        searchOptions={searchOptions}
                        searchModifiers={searchModifiers}
                        searchSuggestions={searchSuggestions}
                        setSearchOptions={setSearchOptions}
                        setSearchModifiers={setSearchModifiers}
                        setSearchSuggestions={setSearchSuggestions}
                        onSearch={onSearch}
                        defaultOption={defaultOption}
                        autoCompleteCategories={['DEPLOYMENTS']}
                    />
                </PageHeader>
                <div className="flex flex-1 relative">
                    <div className="shadow border-primary-300 bg-base-100 w-full overflow-hidden">
                        {renderPanel()}
                    </div>
                    {renderSidePanel()}
                </div>
            </div>
        </section>
    );
};

RiskPage.propTypes = {
    deployments: PropTypes.arrayOf(PropTypes.object).isRequired,
    selectedDeployment: PropTypes.shape({
        id: PropTypes.string.isRequired
    }),
    processGroup: PropTypes.shape({}),
    searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
    searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
    searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
    setSearchOptions: PropTypes.func.isRequired,
    setSearchModifiers: PropTypes.func.isRequired,
    setSearchSuggestions: PropTypes.func.isRequired,
    isViewFiltered: PropTypes.bool.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

RiskPage.defaultProps = {
    selectedDeployment: null,
    processGroup: {}
};

const isViewFiltered = createSelector(
    [selectors.getDeploymentsSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const getSelectedDeployment = (state, props) => {
    const { deploymentId } = props.match.params;
    return deploymentId ? selectors.getSelectedDeployment(state, deploymentId) : null;
};

const getProcessesForDeployment = (state, props) => {
    const { deploymentId } = props.match.params;
    return deploymentId ? selectors.getProcessesByDeployment(state, deploymentId) : {};
};

const mapStateToProps = createStructuredSelector({
    deployments: selectors.getFilteredDeployments,
    selectedDeployment: getSelectedDeployment,
    processGroup: getProcessesForDeployment,
    searchOptions: selectors.getDeploymentsSearchOptions,
    searchModifiers: selectors.getDeploymentsSearchModifiers,
    searchSuggestions: selectors.getDeploymentsSearchSuggestions,
    isViewFiltered
});

const mapDispatchToProps = {
    setSearchOptions: deploymentsActions.setDeploymentsSearchOptions,
    setSearchModifiers: deploymentsActions.setDeploymentsSearchModifiers,
    setSearchSuggestions: deploymentsActions.setDeploymentsSearchSuggestions
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(RiskPage);
