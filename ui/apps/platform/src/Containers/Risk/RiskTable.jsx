import { useState } from 'react';
import PropTypes from 'prop-types';
import { Alert, AlertActionCloseButton, Button } from '@patternfly/react-core';

import NoResultsMessage from 'Components/NoResultsMessage';
import { changeDeploymentRiskPosition, resetAllDeploymentRisks } from 'services/RiskService';
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

    async function handleReorder(deploymentId, aboveDeploymentId, belowDeploymentId) {
        console.log(`[RiskTable] handleReorder called: deploymentId=${deploymentId}, above=${aboveDeploymentId}, below=${belowDeploymentId}`);

        if (isLoading) {
            console.log('[RiskTable] Skipping reorder - already loading');
            return;
        }

        setIsLoading(true);
        setErrorMessage('');
        setSuccessMessage('');

        console.log(`[RiskTable] Calling API with neighbor IDs`);

        try {
            const response = await changeDeploymentRiskPosition(deploymentId, aboveDeploymentId, belowDeploymentId);
            console.log('[RiskTable] API call successful:', response);
            setSuccessMessage(response.message || 'Deployment position updated successfully');
            if (onRefreshData) {
                onRefreshData();
            }
        } catch (error) {
            console.error('[RiskTable] API call failed:', error);
            setErrorMessage(
                error.response?.data?.message ||
                    error.message ||
                    'Failed to update deployment position'
            );
        } finally {
            setIsLoading(false);
        }
    }

    async function handleResetAll() {
        console.log('[RiskTable] handleResetAll called');

        if (isLoading) {
            console.log('[RiskTable] Skipping reset - already loading');
            return;
        }

        setIsLoading(true);
        setErrorMessage('');
        setSuccessMessage('');

        try {
            const response = await resetAllDeploymentRisks();
            console.log('[RiskTable] Reset All API call successful:', response);
            setSuccessMessage(response.message || `Reset ${response.count} deployment risk adjustments`);
            if (onRefreshData) {
                onRefreshData();
            }
        } catch (error) {
            console.error('[RiskTable] Reset All API call failed:', error);
            setErrorMessage(
                error.response?.data?.message ||
                    error.message ||
                    'Failed to reset all deployment risks'
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
            <div className="pf-v5-u-mb-md">
                <Button
                    variant="secondary"
                    onClick={handleResetAll}
                    isDisabled={isLoading}
                >
                    Reset All Risk Adjustments
                </Button>
            </div>
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
