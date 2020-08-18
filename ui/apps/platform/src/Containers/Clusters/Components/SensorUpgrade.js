/* eslint-disable react/jsx-no-bind */
import PropTypes from 'prop-types';
import React from 'react';
import { AlertTriangle, Check, DownloadCloud, Loader, X } from 'react-feather';

import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';

import HealthStatus from './HealthStatus';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';
import { findUpgradeState, formatSensorVersion } from '../cluster.helpers';

const typeStyles = {
    /*
    info: {
        icon: Info,
        bgColor: 'bg-base-200',
        fgColor: 'text-base-600',
    },
    */
    current: {
        Icon: Check,
        bgColor: 'bg-success-200',
        fgColor: 'text-success-700',
    },
    download: {
        Icon: DownloadCloud,
        bgColor: 'bg-tertiary-200',
        fgColor: 'text-tertiary-700',
    },
    progress: {
        Icon: Loader,
        bgColor: 'bg-success-200',
        fgColor: 'text-success-700',
    },
    failure: {
        Icon: X,
        bgColor: 'bg-alert-200',
        fgColor: 'text-alert-700',
    },
    intervention: {
        Icon: AlertTriangle,
        bgColor: 'bg-warning-200',
        fgColor: 'text-warning-700',
    },
};

const trClassName = 'align-top leading-normal';
const thClassName = 'font-600 pl-0 pr-1 py-0 text-left';
const tdClassName = 'p-0 text-left';

/*
 * Sensor Upgrade cell
 * - in Clusters list might have an action (for example, Upgrade available or Retry upgrade)
 * - in Cluster side panel does not have an action (but might have an action in the future)
 */
const SensorUpgrade = ({ upgradeStatus, centralVersion, sensorVersion, isList, actionProps }) => {
    if (upgradeStatus) {
        const upgradeStateObject = findUpgradeState(upgradeStatus);
        if (upgradeStateObject) {
            const { displayValue, type, actionText } = upgradeStateObject;

            let displayElement = null;
            let actionElement = null;

            if (displayValue) {
                const { bgColor, fgColor } = typeStyles[type];
                displayElement = <span className={`${bgColor} ${fgColor}`}>{displayValue}</span>;
            }

            if (actionText) {
                const actionStyle = typeStyles.download;
                if (actionProps) {
                    const { clusterId, upgradeSingleCluster } = actionProps;
                    const onClick = (event) => {
                        event.stopPropagation(); // so click in row does not open side panel
                        upgradeSingleCluster(clusterId);
                    };

                    const { fgColor } = actionStyle;
                    actionElement = (
                        <button
                            type="button"
                            className={`bg-transparent leading-normal m-0 p-0 ${fgColor} underline`}
                            onClick={onClick}
                        >
                            {actionText}
                        </button>
                    );
                } else if (!displayElement) {
                    // Upgrade available is not an action in Cluster side panel,
                    // but it might become an action in the future.
                    const { bgColor, fgColor } = actionStyle;
                    displayElement = <span className={`${bgColor} ${fgColor}`}>{actionText}</span>;
                }
            }

            const upgradeElement = (
                <div>
                    {displayElement}
                    {displayElement && actionElement ? <br /> : null}
                    {actionElement}
                </div>
            );

            const { Icon, bgColor, fgColor } = typeStyles[type];

            // Use table instead of TooltipFieldValue to align version numbers.
            const versionNumbers = (
                <table>
                    <tbody>
                        <tr className={trClassName} key="sensorVersion">
                            <th className={thClassName} scope="row">
                                Sensor version:
                            </th>
                            <td className={tdClassName}>
                                {sensorVersion && type === 'current' ? (
                                    <span className={`${bgColor} ${fgColor}`}>{sensorVersion}</span>
                                ) : (
                                    formatSensorVersion(sensorVersion)
                                )}
                            </td>
                        </tr>
                        <tr className={trClassName} key="centralVersion">
                            <th className={thClassName} scope="row">
                                Central version:
                            </th>
                            <td className={tdClassName}>
                                {type === 'download' ? (
                                    <span className={`${bgColor} ${fgColor}`}>
                                        {centralVersion}
                                    </span>
                                ) : (
                                    centralVersion
                                )}
                            </td>
                        </tr>
                    </tbody>
                </table>
            );

            // Tooltip requires an HTML element instead of a React element as its child :(
            return isList ? (
                <Tooltip content={<TooltipOverlay>{versionNumbers}</TooltipOverlay>}>
                    <div>
                        <HealthStatus Icon={Icon} iconColor={fgColor}>
                            {upgradeElement}
                        </HealthStatus>
                    </div>
                </Tooltip>
            ) : (
                <HealthStatus Icon={Icon} iconColor={fgColor}>
                    <div>
                        {upgradeElement}
                        {versionNumbers}
                    </div>
                </HealthStatus>
            );
        }
    }

    return <HealthStatusNotApplicable />;
};

SensorUpgrade.propTypes = {
    // Document the properties accessed by the helper function:
    upgradeStatus: PropTypes.shape({
        upgradability: PropTypes.string,
        mostRecentProcess: PropTypes.shape({
            active: PropTypes.bool,
            process: PropTypes.shape({
                upgradeState: PropTypes.string,
            }),
        }),
    }),
    sensorVersion: PropTypes.string,
    centralVersion: PropTypes.string.isRequired,
    isList: PropTypes.bool.isRequired,
    actionProps: PropTypes.shape({
        clusterId: PropTypes.string.isRequired,
        upgradeSingleCluster: PropTypes.func.isRequired,
    }),
};

SensorUpgrade.defaultProps = {
    upgradeStatus: null,
    sensorVersion: '',
    actionProps: null,
};

export default SensorUpgrade;
