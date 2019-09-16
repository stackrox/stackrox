import React from 'react';
import PropTypes from 'prop-types';

import { resolveAlert } from 'services/AlertsService';
import { whitelistDeployments } from 'services/PoliciesService';

import Tooltip from 'rc-tooltip';
import * as Icon from 'react-feather';

function ViolationActionButtons({ violation, setSelectedAlertId }) {
    function resolveAlertAction(whitelist) {
        const unselectAlert = () => setSelectedAlertId(null);
        return e => {
            e.stopPropagation();
            resolveAlert(violation.id, whitelist).then(unselectAlert, unselectAlert);
        };
    }

    function whitelistDeploymentAction(e) {
        e.stopPropagation();
        whitelistDeployments(violation.policy.id, [violation.deployment.name]);
    }

    const isRuntimeAlert = violation && violation.lifecycleStage === 'RUNTIME';
    return (
        <div
            data-test-id="alerts-hover-actions"
            className="flex border-2 border-r-2 border-base-400 bg-base-100 shadow"
        >
            {isRuntimeAlert && (
                <div className="flex">
                    <Tooltip
                        placement="top"
                        mouseLeaveDelay={0}
                        overlay={
                            <div>Resolve violation and add affected processes to whitelist</div>
                        }
                        overlayClassName="pointer-events-none"
                    >
                        <button
                            type="button"
                            data-test-id="resolve-button"
                            className="p-1 px-4 hover:bg-primary-200 text-primary-600 hover:text-primary-700"
                            onClick={resolveAlertAction(true)}
                        >
                            <Icon.ShieldOff className="mt-1 h-4 w-4" />
                        </button>
                    </Tooltip>
                    <Tooltip
                        placement="top"
                        mouseLeaveDelay={0}
                        overlay={<div>Mark as resolved</div>}
                        overlayClassName="pointer-events-none"
                    >
                        <button
                            type="button"
                            data-test-id="resolve-button"
                            className="p-1 px-4 hover:bg-primary-200 text-primary-600 hover:text-primary-700 border-l-2 border-base-400"
                            onClick={resolveAlertAction(false)}
                        >
                            <Icon.Check className="mt-1 h-4 w-4" />
                        </button>
                    </Tooltip>
                </div>
            )}
            <Tooltip
                placement="top"
                mouseLeaveDelay={0}
                overlay={
                    <div>
                        Whitelist <br /> deployment
                    </div>
                }
                overlayClassName="pointer-events-none text-center"
            >
                <button
                    data-test-id="whitelist-deployment-button"
                    type="button"
                    className={`p-1 px-4 hover:bg-primary-200 text-primary-600 hover:text-primary-700 ${
                        isRuntimeAlert ? 'border-l-2 border-base-400' : ''
                    }`}
                    onClick={whitelistDeploymentAction}
                >
                    <Icon.BellOff className="mt-1 h-4 w-4" />
                </button>
            </Tooltip>
        </div>
    );
}

ViolationActionButtons.propTypes = {
    violation: PropTypes.shape({
        id: PropTypes.string.isRequired,
        lifecycleStage: PropTypes.string.isRequired,
        deployment: PropTypes.shape({
            name: PropTypes.string.isRequired
        }).isRequired,
        policy: PropTypes.shape({
            id: PropTypes.string.isRequired
        }).isRequired
    }).isRequired,
    setSelectedAlertId: PropTypes.func.isRequired
};

export default ViolationActionButtons;
