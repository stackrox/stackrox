import React, { useContext, useState, useCallback } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import useDeepCompareEffect from 'use-deep-compare-effect';

import TableHeader from 'Components/TableHeader';
import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import { DEFAULT_PAGE_SIZE } from 'Components/Table';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import workflowStateContext from 'Containers/workflowStateContext';
import { fetchDeployments, fetchDeploymentsCount } from 'services/DeploymentsService';
import RiskTable from './RiskTable';
import {
    filterAllowedSearch,
    convertToRestSearch,
    convertSortToGraphQLFormat,
    convertSortToRestFormat,
} from './riskPageUtils';

const DEFAULT_RISK_SORT = [{ id: 'Priority', desc: false }];
function RiskTablePanel({
    history,
    selectedDeploymentId,
    setSelectedDeploymentId,
    isViewFiltered,
    setIsViewFiltered,
    searchOptions,
}) {
    const workflowState = useContext(workflowStateContext);
    const pageSearch = workflowState.search[searchParams.page];
    const sortOption = workflowState.sort[sortParams.page] || DEFAULT_RISK_SORT;
    const currentPage = workflowState.paging[pagingParams.page];

    const [currentDeployments, setCurrentDeployments] = useState([]);
    const [deploymentCount, setDeploymentsCount] = useState(0);

    function setPage(newPage) {
        history.push(workflowState.setPage(newPage).toUrl());
    }
    const setSortOption = useCallback(
        (newSortOption) => {
            const convertedSortOption = convertSortToGraphQLFormat(newSortOption);

            const newUrl = workflowState.setSort(convertedSortOption).toUrl();

            history.push(newUrl);
        },
        [history, workflowState]
    );

    useDeepCompareEffect(() => {
        const filteredSearch = filterAllowedSearch(searchOptions, pageSearch || {});
        const restSearch = convertToRestSearch(filteredSearch || {});
        const restSort = convertSortToRestFormat(sortOption);

        fetchDeployments(restSearch, restSort, currentPage, DEFAULT_PAGE_SIZE).then(
            setCurrentDeployments
        );
        fetchDeploymentsCount(restSearch).then(setDeploymentsCount);

        if (restSearch.length) {
            setIsViewFiltered(true);
        } else {
            setIsViewFiltered(false);
        }
    }, [pageSearch, sortOption, currentPage, searchOptions]);

    const paginationComponent = (
        <TablePagination
            page={currentPage}
            dataLength={deploymentCount}
            pageSize={DEFAULT_PAGE_SIZE}
            setPage={setPage}
        />
    );

    const headerComponent = (
        <TableHeader length={deploymentCount} type="Deployment" isViewFiltered={isViewFiltered} />
    );
    return (
        <Panel headerTextComponent={headerComponent} headerComponents={paginationComponent}>
            <div className="h-full w-full">
                <RiskTable
                    currentDeployments={currentDeployments}
                    setSelectedDeploymentId={setSelectedDeploymentId}
                    selectedDeploymentId={selectedDeploymentId}
                    setSortOption={setSortOption}
                />
            </div>
        </Panel>
    );
}

RiskTablePanel.propTypes = {
    history: ReactRouterPropTypes.history.isRequired,
    selectedDeploymentId: PropTypes.string,
    setSelectedDeploymentId: PropTypes.func.isRequired,
    isViewFiltered: PropTypes.bool.isRequired,
    setIsViewFiltered: PropTypes.func.isRequired,
    searchOptions: PropTypes.arrayOf(PropTypes.string),
};

RiskTablePanel.defaultProps = {
    selectedDeploymentId: null,
    searchOptions: [],
};

export default withRouter(RiskTablePanel);
