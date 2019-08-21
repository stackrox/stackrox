import React from 'react';
import PropTypes from 'prop-types';

import { PageHeaderComponent } from 'Components/PageHeader';
import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePaginationV2';
import { pageSize } from 'Components/Table';

import RiskTable from './RiskTable';

function RiskTablePanel({
    currentDeployments,
    currentPage,
    setCurrentPage,
    deploymentCount,
    selectedDeploymentId,
    setSelectedDeploymentId,
    setSortOption,
    isViewFiltered
}) {
    const pageCount = Math.ceil(deploymentCount / pageSize);
    const paginationComponent = (
        <TablePagination page={currentPage} pageCount={pageCount} setPage={setCurrentPage} />
    );

    const headerComponent = (
        <PageHeaderComponent
            length={deploymentCount}
            type="Deployment"
            isViewFiltered={isViewFiltered}
        />
    );
    return (
        <Panel headerTextComponent={headerComponent} headerComponents={paginationComponent}>
            <div className="w-full">
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
    currentDeployments: PropTypes.arrayOf(PropTypes.object).isRequired,
    currentPage: PropTypes.number.isRequired,
    setCurrentPage: PropTypes.func.isRequired,
    deploymentCount: PropTypes.number.isRequired,
    selectedDeploymentId: PropTypes.string,
    setSelectedDeploymentId: PropTypes.func.isRequired,
    setSortOption: PropTypes.func.isRequired,
    isViewFiltered: PropTypes.bool.isRequired
};

RiskTablePanel.defaultProps = {
    selectedDeploymentId: undefined
};

export default RiskTablePanel;
