import { useState } from 'react';
import PropTypes from 'prop-types';
import { Alert, AlertActionCloseButton } from '@patternfly/react-core';

import NoResultsMessage from 'Components/NoResultsMessage';
import { changeDeploymentRiskPosition } from 'services/RiskService';
import DraggableRiskTable from './DraggableRiskTable';

function RiskTable({
    currentDeployments,
    setSelectedDeploymentId,
    selectedDeploymentId,
    onRefreshData,
}) {
    const [successMessage, setSuccessMessage] = useState('');
    const [errorMessage, setErrorMessage] = useState('');
    const [isLoading, setIsLoading] = useState(false);

    function updateSelectedDeployment(row) {
        setSelectedDeploymentId(row.deployment.id);
    }

    async function handleReorder(fromIndex, toIndex) {
        if (fromIndex === toIndex || isLoading) {
            return;
        }

        setIsLoading(true);
        setErrorMessage('');
        setSuccessMessage('');

        const movedRow = currentDeployments[fromIndex];
        const deploymentId = movedRow.deployment.id;

        // Determine direction based on indices
        const direction = fromIndex < toIndex ? 'RISK_POSITION_DOWN' : 'RISK_POSITION_UP';

        try {
            const response = await changeDeploymentRiskPosition(deploymentId, direction);
            setSuccessMessage(response.message || 'Deployment position updated successfully');
            if (onRefreshData) {
                onRefreshData();
            }
        } catch (error) {
            setErrorMessage(
                error.response?.data?.message ||
                    error.message ||
                    'Failed to update deployment position'
            );
        } finally {
            setIsLoading(false);
        }
    }

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
            <DraggableRiskTable
                currentDeployments={currentDeployments}
                onRowClick={updateSelectedDeployment}
                selectedDeploymentId={selectedDeploymentId}
                onReorder={handleReorder}
            />
        </>
    );
}

RiskTable.propTypes = {
    currentDeployments: PropTypes.arrayOf(PropTypes.object).isRequired,
    selectedDeploymentId: PropTypes.string,
    setSelectedDeploymentId: PropTypes.func.isRequired,
    onRefreshData: PropTypes.func,
};

RiskTable.defaultProps = {
    selectedDeploymentId: undefined,
    onRefreshData: undefined,
};

export default RiskTable;
