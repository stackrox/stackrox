import React from 'react';
import PropTypes from 'prop-types';

import { Tooltip, TooltipOverlay, DetailedTooltipOverlay } from '@stackrox/ui-components';
import { healthStatusLabels } from 'messages/common';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';

import HealthStatus from './HealthStatus';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';
import {
    delayedAdmissionControlStatusStyle,
    healthStatusStyles,
    isDelayedSensorHealthStatus,
} from '../cluster.helpers';

const trClassName = 'align-bottom leading-normal'; // align-bottom in case heading text wraps
const thClassName = 'font-600 pl-0 pr-1 py-0 text-left';
const tdClassName = 'p-0 text-right';
const tdErrorsClassName = 'font-600 pb-0 pl-0 pr-1 pt-2 text-left'; // pt for gap above errors

const testId = 'admissionControlStatus';

/*
 * Admission Control Status in Clusters list if `isList={true}` or Cluster side panel if `isList={false}`
 *
 * Caller is responsible for optional chaining in case healthStatus is null.
 */
const AdmissionControlStatus = ({ healthStatus, currentDatetime, isList }) => {
    if (healthStatus?.admissionControlHealthStatus) {
        const {
            admissionControlHealthStatus,
            admissionControlHealthInfo,
            healthInfoComplete,
            sensorHealthStatus,
            lastContact,
        } = healthStatus;
        const { Icon, bgColor, fgColor } =
            lastContact && isDelayedSensorHealthStatus(sensorHealthStatus)
                ? delayedAdmissionControlStatusStyle
                : healthStatusStyles[admissionControlHealthStatus];
        const labelElement = (
            <span className={`${bgColor} ${fgColor}`}>
                {healthStatusLabels[admissionControlHealthStatus]}
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

        if (admissionControlHealthInfo) {
            const notAvailable = 'n/a';
            const { totalReadyPods, totalDesiredPods, statusErrors } = admissionControlHealthInfo;
            const totalsElement = (
                <table data-testid="admissionControlHealthInfo">
                    <tbody>
                        <tr className={trClassName} key="totalReadyPods">
                            <th className={thClassName} scope="row">
                                Admission Control pods ready:
                            </th>
                            <td className={tdClassName} data-testid="totalReadyPods">
                                <span className={`${bgColor} ${fgColor}`}>
                                    {totalReadyPods == null ? notAvailable : totalReadyPods}
                                </span>
                            </td>
                        </tr>
                        <tr className={trClassName} key="totalDesiredPods">
                            <th className={thClassName} scope="row">
                                Admission Control pods expected:
                            </th>
                            <td className={tdClassName} data-testid="totalDesiredPods">
                                {totalDesiredPods == null ? notAvailable : totalDesiredPods}
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
                    <div data-testid="admissionControlInfoComplete">
                        <strong>Upgrade Sensor</strong> to get complete Admission Control health
                        information
                    </div>
                </div>
            );

            return isList ? (
                <Tooltip
                    content={
                        <DetailedTooltipOverlay
                            title="Admission Control Health Information"
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

        if (admissionControlHealthStatus === 'UNAVAILABLE') {
            const reasonUnavailable = (
                <div data-testid="admissionControlInfoComplete">
                    <strong>Upgrade Sensor</strong> to get Admission Control health information
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

AdmissionControlStatus.propTypes = {
    healthStatus: PropTypes.shape({
        admissionControlHealthStatus: PropTypes.oneOf([
            'UNINITIALIZED',
            'UNAVAILABLE',
            'UNHEALTHY',
            'DEGRADED',
            'HEALTHY',
        ]),
        admissionControlHealthInfo: PropTypes.shape({
            totalDesiredPods: PropTypes.number,
            totalReadyPods: PropTypes.number,
            statusErrors: PropTypes.arrayOf(PropTypes.string),
        }),
        healthInfoComplete: PropTypes.bool,
        sensorHealthStatus: PropTypes.oneOf(['UNINITIALIZED', 'UNHEALTHY', 'DEGRADED', 'HEALTHY']),
        lastContact: PropTypes.string, // ISO 8601
    }),
    currentDatetime: PropTypes.instanceOf(Date).isRequired,
    isList: PropTypes.bool.isRequired,
};

AdmissionControlStatus.defaultProps = {
    healthStatus: null,
};

export default AdmissionControlStatus;
