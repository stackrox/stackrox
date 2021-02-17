import React from 'react';
import PropTypes from 'prop-types';

import { resolveAlert } from 'services/AlertsService';
import { excludeDeployments } from 'services/PoliciesService';

import * as Icon from 'react-feather';
import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';
import { ENFORCEMENT_ACTIONS } from 'constants/enforcementActions';
import VIOLATION_STATES from 'constants/violationStates';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';

function ViolationActionButtons({ violation, setSelectedAlertId }) {
    function resolveAlertAction(addToBaseline) {
        const unselectAlert = () => setSelectedAlertId(null);
        return (e) => {
            e.stopPropagation();
            resolveAlert(violation.id, addToBaseline).then(unselectAlert, unselectAlert);
        };
    }

    function excludeDeploymentAction(e) {
        e.stopPropagation();
        excludeDeployments(violation.policy.id, [violation.deployment.name]);
    }

    const isAttemptedViolation = violation?.state === VIOLATION_STATES.ATTEMPTED;
    const isResolved = violation?.state === VIOLATION_STATES.RESOLVED;
    const isRuntimeAlert = violation?.lifecycleStage === LIFECYCLE_STAGES.RUNTIME;
    const isDeployCreateAttemptedAlert =
        violation?.enforcementAction === ENFORCEMENT_ACTIONS.FAIL_DEPLOYMENT_CREATE_ENFORCEMENT;

    return (
        <div
            data-testid="alerts-hover-actions"
            className="flex border-2 border-r-2 border-base-400 bg-base-100 shadow"
        >
            {!isResolved && (
                <div className="flex">
                    {isRuntimeAlert && (
                        <Tooltip
                            content={
                                <TooltipOverlay>
                                    Resolve violation and add affected processes to excluded scope
                                </TooltipOverlay>
                            }
                        >
                            <button
                                type="button"
                                data-testid="resolve-button"
                                className="p-1 px-4 hover:bg-primary-200 text-primary-600 hover:text-primary-700"
                                onClick={resolveAlertAction(true)}
                            >
                                <Icon.ShieldOff className="my-1 h-4 w-4" />
                            </button>
                        </Tooltip>
                    )}
                    {(isRuntimeAlert || isAttemptedViolation) && (
                        <Tooltip content={<TooltipOverlay>Mark as resolved</TooltipOverlay>}>
                            <button
                                type="button"
                                data-testid="resolve-button"
                                className="p-1 px-4 hover:bg-primary-200 text-primary-600 hover:text-primary-700 border-l-2 border-base-400"
                                onClick={resolveAlertAction(false)}
                            >
                                <Icon.Check className="my-1 h-4 w-4" />
                            </button>
                        </Tooltip>
                    )}
                </div>
            )}
            {!isDeployCreateAttemptedAlert && (
                <Tooltip content={<TooltipOverlay>Exclude deployment</TooltipOverlay>}>
                    <button
                        data-testid="exclude-deployment-button"
                        type="button"
                        className={`p-1 px-4 hover:bg-primary-200 text-primary-600 hover:text-primary-700 ${
                            isRuntimeAlert ? 'border-l-2 border-base-400' : ''
                        }`}
                        onClick={excludeDeploymentAction}
                    >
                        <Icon.BellOff className="my-1 h-4 w-4" />
                    </button>
                </Tooltip>
            )}
        </div>
    );
}

ViolationActionButtons.propTypes = {
    violation: PropTypes.shape({
        id: PropTypes.string.isRequired,
        lifecycleStage: PropTypes.string.isRequired,
        deployment: PropTypes.shape({
            name: PropTypes.string.isRequired,
        }).isRequired,
        policy: PropTypes.shape({
            id: PropTypes.string.isRequired,
        }).isRequired,
        state: PropTypes.string.isRequired,
        enforcementAction: PropTypes.string.isRequired,
    }).isRequired,
    setSelectedAlertId: PropTypes.func.isRequired,
};

export default ViolationActionButtons;
