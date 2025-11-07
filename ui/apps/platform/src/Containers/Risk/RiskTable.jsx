import { useState } from 'react';
import PropTypes from 'prop-types';
import { Alert, AlertActionCloseButton } from '@patternfly/react-core';

import NoResultsMessage from 'Components/NoResultsMessage';
import Table from 'Components/TableV2';
import { changeDeploymentRiskPosition } from 'services/RiskService';

import riskTableColumnDescriptors from './riskTableColumnDescriptors';

function sortOptionFromTableState(state) {
    let sortOption = null;
    if (state.sorted.length && state.sorted[0].id) {
        const column = riskTableColumnDescriptors.find(
            (col) => col.accessor === state.sorted[0].id
        );
        sortOption = {
            field: column.searchField,
            reversed: state.sorted[0].desc,
        };
    }
    return sortOption;
}

function RiskTable({
    currentDeployments,
    setSelectedDeploymentId,
    selectedDeploymentId,
    setSortOption,
    onRefreshData,
}) {
    const [loadingDeploymentId, setLoadingDeploymentId] = useState(null);
    const [successMessage, setSuccessMessage] = useState('');
    const [errorMessage, setErrorMessage] = useState('');

    function onFetchData(state) {
        const newSortOption = sortOptionFromTableState(state);
        if (!newSortOption) {
            return;
        }
        setSortOption(newSortOption);
    }

    function updateSelectedDeployment({ deployment }) {
        setSelectedDeploymentId(deployment.id);
    }

    async function handleMoveUp(deploymentId) {
        setLoadingDeploymentId(deploymentId);
        setErrorMessage('');
        setSuccessMessage('');

        try {
            const response = await changeDeploymentRiskPosition(deploymentId, 'RISK_POSITION_UP');
            setSuccessMessage(response.message || 'Deployment moved up successfully');
            if (onRefreshData) {
                onRefreshData();
            }
        } catch (error) {
            setErrorMessage(error.response?.data?.message || error.message || 'Failed to move deployment up');
        } finally {
            setLoadingDeploymentId(null);
        }
    }

    async function handleMoveDown(deploymentId) {
        setLoadingDeploymentId(deploymentId);
        setErrorMessage('');
        setSuccessMessage('');

        try {
            const response = await changeDeploymentRiskPosition(deploymentId, 'RISK_POSITION_DOWN');
            setSuccessMessage(response.message || 'Deployment moved down successfully');
            if (onRefreshData) {
                onRefreshData();
            }
        } catch (error) {
            setErrorMessage(error.response?.data?.message || error.message || 'Failed to move deployment down');
        } finally {
            setLoadingDeploymentId(null);
        }
    }

    // Create enhanced column descriptors with handlers
    const columnsWithHandlers = riskTableColumnDescriptors.map((col) => {
        if (col.accessor === 'actions') {
            return {
                ...col,
                Cell: (props) => col.Cell({
                    ...props,
                    onMoveUp: handleMoveUp,
                    onMoveDown: handleMoveDown,
                    loadingDeploymentId,
                }),
            };
        }
        return col;
    });

    if (!currentDeployments.length) {
        return <NoResultsMessage message="No results found. Please refine your search." />;
    }

    return (
        <>
            {successMessage && (
                <Alert
                    variant="success"
                    title={successMessage}
                    actionClose={<AlertActionCloseButton onClose={() => setSuccessMessage('')} />}
                    className="pf-v5-u-mb-md"
                />
            )}
            {errorMessage && (
                <Alert
                    variant="danger"
                    title={errorMessage}
                    actionClose={<AlertActionCloseButton onClose={() => setErrorMessage('')} />}
                    className="pf-v5-u-mb-md"
                />
            )}
            <Table
                idAttribute="deployment.id"
                rows={currentDeployments}
                columns={columnsWithHandlers}
                onRowClick={updateSelectedDeployment}
                selectedRowId={selectedDeploymentId}
                onFetchData={onFetchData}
                noDataText="No results found. Please refine your search."
            />
        </>
    );
}

RiskTable.propTypes = {
    currentDeployments: PropTypes.arrayOf(PropTypes.object).isRequired,
    selectedDeploymentId: PropTypes.string,
    setSelectedDeploymentId: PropTypes.func.isRequired,
    setSortOption: PropTypes.func.isRequired,
    onRefreshData: PropTypes.func,
};

RiskTable.defaultProps = {
    selectedDeploymentId: undefined,
    onRefreshData: undefined,
};

export default RiskTable;
