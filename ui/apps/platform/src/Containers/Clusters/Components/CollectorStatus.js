import React from 'react';
import PropTypes from 'prop-types';

import { Tooltip, TooltipOverlay, DetailedTooltipOverlay } from '@stackrox/ui-components';
import { healthStatusLabels } from 'messages/common';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';

import HealthStatus from './HealthStatus';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';
import {
    delayedCollectorStatusStyle,
    healthStatusStyles,
    isDelayedSensorHealthStatus,
} from '../cluster.helpers';

const trClassName = 'align-bottom leading-normal'; // align-bottom in case heading text wraps
const thClassName = 'font-600 pl-0 pr-1 py-0 text-left';
const tdClassName = 'p-0 text-right';
const tdErrorsClassName = 'font-600 pb-0 pl-0 pr-1 pt-2 text-left'; // pt for gap above errors

const testId = 'collectorStatus';

/*
 * Collector Status in Clusters list if `isList={true}` or Cluster side panel if `isList={false}`
 *
 * Caller is responsible for optional chaining in case healthStatus is null.
 */
const CollectorStatus = ({ healthStatus, currentDatetime, isList }) => {
    if (healthStatus?.collectorHealthStatus) {
        const {
            collectorHealthStatus,
            collectorHealthInfo,
            healthInfoComplete,
            sensorHealthStatus,
            lastContact,
        } = healthStatus;
        const { Icon, bgColor, fgColor } =
            lastContact && isDelayedSensorHealthStatus(sensorHealthStatus)
                ? delayedCollectorStatusStyle
                : healthStatusStyles[collectorHealthStatus];
        const labelElement = (
            <span className={`${bgColor} ${fgColor}`}>
                {healthStatusLabels[collectorHealthStatus]}
            </span>
        );

        // In rare case that the block does not fit in a narrow column,
        // the space and "whitespace-nowrap" cause time phrase to wrap as a unit.
        // Order arguments according to date-fns@2 convention:
        // If lastContact <= currentDateTime: X units ago
        const statusElement =
            lastContact && isDelayedSensorHealthStatus(sensorHealthStatus) ? (
                <div data-testid={testId}>
                    {labelElement}{' '}
                    <span className="whitespace-nowrap">
                        {getDistanceStrictAsPhrase(lastContact, currentDatetime)}
                    </span>
                </div>
            ) : (
                <div data-testid={testId}>{labelElement}</div>
            );

        if (collectorHealthInfo) {
            const notAvailable = 'n/a';
            const {
                totalReadyPods,
                totalDesiredPods,
                totalRegisteredNodes,
                version,
                statusErrors,
            } = collectorHealthInfo;
            const totalsElement = (
                <table>
                    <tbody>
                        <tr className={trClassName} key="version">
                            <th className={thClassName} scope="row">
                                Collector version:
                            </th>
                            <td className={tdClassName} data-testid="version">
                                {version || notAvailable}
                            </td>
                        </tr>
                        <tr className={trClassName} key="totalReadyPods">
                            <th className={thClassName} scope="row">
                                Collector pods ready:
                            </th>
                            <td className={tdClassName} data-testid="totalReadyPods">
                                <span className={`${bgColor} ${fgColor}`}>
                                    {totalReadyPods == null ? notAvailable : totalReadyPods}
                                </span>
                            </td>
                        </tr>
                        <tr className={trClassName} key="totalDesiredPods">
                            <th className={thClassName} scope="row">
                                Collector pods expected:
                            </th>
                            <td className={tdClassName} data-testid="totalDesiredPods">
                                <span className={`${bgColor} ${fgColor}`}>
                                    {totalDesiredPods == null ? notAvailable : totalDesiredPods}
                                </span>
                            </td>
                        </tr>
                        <tr className={trClassName} key="totalRegisteredNodes">
                            <th className={thClassName} scope="row">
                                Registered nodes in cluster:
                            </th>
                            <td className={tdClassName} data-testid="totalRegisteredNodes">
                                {totalRegisteredNodes == null ? notAvailable : totalRegisteredNodes}
                            </td>
                        </tr>
                        {statusErrors && statusErrors.length > 0 && (
                            <tr className={trClassName} key="statusErrors">
                                <td
                                    className={tdErrorsClassName}
                                    colSpan={2}
                                    data-testid="statusErrors"
                                >
                                    <ul>
                                        {statusErrors.map((err) => (
                                            <li key={err}>
                                                <span
                                                    className={`${healthStatusStyles.UNHEALTHY.fgColor} break-all`}
                                                >
                                                    {err}
                                                </span>
                                            </li>
                                        ))}
                                    </ul>
                                </td>
                            </tr>
                        )}
                    </tbody>
                </table>
            );

            const infoElement = healthInfoComplete ? (
                totalsElement
            ) : (
                <div>
                    {totalsElement}
                    <div data-testid="healthInfoComplete">
                        <strong>Upgrade Sensor</strong> to get complete Collector health information
                    </div>
                </div>
            );

            return isList ? (
                <Tooltip
                    content={
                        <DetailedTooltipOverlay
                            title="Collector Health Information"
                            body={infoElement}
                        />
                    }
                >
                    <div>
                        <HealthStatus Icon={Icon} iconColor={fgColor}>
                            {statusElement}
                        </HealthStatus>
                    </div>
                </Tooltip>
            ) : (
                <HealthStatus Icon={Icon} iconColor={fgColor}>
                    <div>
                        {statusElement}
                        {infoElement}
                    </div>
                </HealthStatus>
            );
        }

        if (collectorHealthStatus === 'UNAVAILABLE') {
            const reasonUnavailable = (
                <div data-testid="healthInfoComplete">
                    <strong>Upgrade Sensor</strong> to get Collector health information
                </div>
            );

            return isList ? (
                <Tooltip content={<TooltipOverlay>{reasonUnavailable}</TooltipOverlay>}>
                    <div>
                        <HealthStatus Icon={Icon} iconColor={fgColor}>
                            {statusElement}
                        </HealthStatus>
                    </div>
                </Tooltip>
            ) : (
                <HealthStatus Icon={Icon} iconColor={fgColor}>
                    <div>
                        {statusElement}
                        {reasonUnavailable}
                    </div>
                </HealthStatus>
            );
        }

        // UNINITIALIZED
        return (
            <HealthStatus Icon={Icon} iconColor={fgColor}>
                <div>{statusElement}</div>
            </HealthStatus>
        );
    }

    return <HealthStatusNotApplicable testId={testId} />;
};

CollectorStatus.propTypes = {
    healthStatus: PropTypes.shape({
        collectorHealthStatus: PropTypes.oneOf([
            'UNINITIALIZED',
            'UNAVAILABLE',
            'UNHEALTHY',
            'DEGRADED',
            'HEALTHY',
        ]),
        collectorHealthInfo: PropTypes.shape({
            version: PropTypes.string,
            totalDesiredPods: PropTypes.number,
            totalReadyPods: PropTypes.number,
            totalRegisteredNodes: PropTypes.number,
            statusErrors: PropTypes.arrayOf(PropTypes.string),
        }),
        healthInfoComplete: PropTypes.bool,
        sensorHealthStatus: PropTypes.oneOf(['UNINITIALIZED', 'UNHEALTHY', 'DEGRADED', 'HEALTHY']),
        lastContact: PropTypes.string, // ISO 8601
    }),
    currentDatetime: PropTypes.instanceOf(Date).isRequired,
    isList: PropTypes.bool.isRequired,
};

CollectorStatus.defaultProps = {
    healthStatus: null,
};

export default CollectorStatus;
