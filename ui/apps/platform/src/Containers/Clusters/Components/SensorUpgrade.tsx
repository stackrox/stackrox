import React, { ReactElement } from 'react';
import { Tooltip } from '@patternfly/react-core';

import HealthStatus from './HealthStatus';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';
import { findUpgradeState, formatSensorVersion, sensorUpgradeStylesPF } from '../cluster.helpers';
import { SensorUpgradeStatus } from '../clusterTypes';

const trClassName = 'align-top leading-normal';
const thClassName = 'font-600 pl-0 pr-1 py-0 text-left';
const tdClassName = 'p-0 text-left';

const testId = 'sensorUpgrade';

/*
 * Sensor Upgrade cell
 * - in Clusters list might have an action (for example, Upgrade available or Retry upgrade)
 * - in Cluster side panel does not have an action (but might have an action in the future)
 */

type SensorUpgradeProps = {
    upgradeStatus?: SensorUpgradeStatus;
    centralVersion: string;
    sensorVersion?: string;
    isList?: boolean;
    actionProps?: {
        clusterId: string;
        upgradeSingleCluster: (clusterId) => void;
    };
};

function SensorUpgrade({
    upgradeStatus,
    centralVersion,
    sensorVersion = '',
    isList = false,
    actionProps,
}: SensorUpgradeProps): ReactElement {
    if (upgradeStatus) {
        const upgradeStateObject = findUpgradeState(upgradeStatus);
        if (upgradeStateObject) {
            const { displayValue, type, actionText } = upgradeStateObject;

            let displayElement: ReactElement | null = null;
            let actionElement: ReactElement | null = null;

            if (displayValue) {
                const { bgColor, fgColor } = sensorUpgradeStylesPF[type];
                displayElement = (
                    <span className={`${bgColor as string} ${fgColor as string}`}>
                        {displayValue}
                    </span>
                );
            }

            if (actionText) {
                const actionStyle = sensorUpgradeStylesPF.download;
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
                <div data-testid={testId}>
                    {displayElement}
                    {displayElement && actionElement && <br />}
                    {actionElement}
                </div>
            );

            const { Icon, bgColor, fgColor } = sensorUpgradeStylesPF[type];
            const icon = <Icon className="h-4 w-4" />;

            // Use table instead of TooltipFieldValue to align version numbers.
            const versionNumbers = (
                <table>
                    <tbody>
                        <tr className={trClassName} key="sensorVersion">
                            <th className={thClassName} scope="row">
                                Sensor version:
                            </th>
                            <td className={tdClassName} data-testid="sensorVersion">
                                {sensorVersion && type === 'current' ? (
                                    <span className={`${bgColor as string} ${fgColor as string}`}>
                                        {sensorVersion}
                                    </span>
                                ) : (
                                    formatSensorVersion(sensorVersion)
                                )}
                            </td>
                        </tr>
                        <tr className={trClassName} key="centralVersion">
                            <th className={thClassName} scope="row">
                                Central version:
                            </th>
                            <td className={tdClassName} data-testid="centralVersion">
                                {type === 'download' ? (
                                    <span className={`${bgColor as string} ${fgColor as string}`}>
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

            let detailMessage = '';
            if (type === 'failure') {
                detailMessage = upgradeStatus?.mostRecentProcess?.progress?.upgradeStatusDetail;
            } else if (type === 'intervention') {
                detailMessage = upgradeStatus?.upgradabilityStatusReason;
            }

            const detailElement = detailMessage ? (
                <div className="mb-2" data-testid="upgradeStatusDetail">
                    {detailMessage}
                </div>
            ) : null;

            if (isList) {
                const overlayElement = detailElement ? (
                    <div>
                        {detailElement}
                        {versionNumbers}
                    </div>
                ) : (
                    versionNumbers
                );

                return (
                    <Tooltip content={overlayElement}>
                        <div>
                            <HealthStatus icon={icon} iconColor={fgColor}>
                                {upgradeElement}
                            </HealthStatus>
                        </div>
                    </Tooltip>
                );
            }

            return (
                <HealthStatus icon={icon} iconColor={fgColor}>
                    <div>
                        {upgradeElement}
                        {detailElement}
                        {versionNumbers}
                    </div>
                </HealthStatus>
            );
        }
    }

    return <HealthStatusNotApplicable testId={testId} />;
}

export default SensorUpgrade;
