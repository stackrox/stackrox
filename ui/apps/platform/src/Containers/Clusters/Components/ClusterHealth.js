import PropTypes from 'prop-types';
import React from 'react';

import ClusterStatus from './ClusterStatus';
import CollectorStatus from './CollectorStatus';
import CredentialExpiration from './CredentialExpiration';
import SensorStatus from './SensorStatus';
import SensorUpgrade from './SensorUpgrade';

const trClass = 'align-top leading-normal';
const thClass = 'pl-0 pr-2 py-1 text-left whitespace-no-wrap';
const tdClass = 'px-0 py-1';

/*
 * Cluster Health in Cluster side panel.
 *
 * The child elements assume that this component is responsible for optional chaining.
 */
const ClusterHealth = ({ healthStatus, status, centralVersion, currentDatetime }) => {
    return (
        <table>
            <tbody>
                <tr className={trClass} key="Cluster Status">
                    <th className={thClass} scope="row">
                        Cluster Status
                    </th>
                    <td className={tdClass}>
                        <ClusterStatus overallHealthStatus={healthStatus?.overallHealthStatus} />
                    </td>
                </tr>
                <tr className={trClass} key="Sensor Status">
                    <th className={thClass} scope="row">
                        Sensor Status
                    </th>
                    <td className={tdClass}>
                        <SensorStatus
                            healthStatus={healthStatus}
                            currentDatetime={currentDatetime}
                        />
                    </td>
                </tr>
                <tr className={trClass} key="Collector Status">
                    <th className={thClass} scope="row">
                        Collector Status
                    </th>
                    <td className={tdClass}>
                        <CollectorStatus
                            healthStatus={healthStatus}
                            currentDatetime={currentDatetime}
                            isList={false}
                        />
                    </td>
                </tr>
                <tr className={trClass} key="Sensor Upgrade">
                    <th className={thClass} scope="row">
                        Sensor Upgrade
                    </th>
                    <td className={tdClass}>
                        <SensorUpgrade
                            upgradeStatus={status?.upgradeStatus}
                            sensorVersion={status?.sensorVersion}
                            centralVersion={centralVersion}
                            isList={false}
                            actionProps={null}
                        />
                    </td>
                </tr>
                <tr className={trClass} key="Credential Expiration">
                    <th className={thClass} scope="row">
                        Credential Expiration
                    </th>
                    <td className={tdClass}>
                        <CredentialExpiration
                            certExpiryStatus={status?.certExpiryStatus}
                            currentDatetime={currentDatetime}
                            isList={false}
                        />
                    </td>
                </tr>
            </tbody>
        </table>
    );
};

ClusterHealth.propTypes = {
    healthStatus: PropTypes.shape({
        collectorHealthStatus: PropTypes.string,
        collectorHealthInfo: PropTypes.shape({
            totalDesiredPods: PropTypes.number.isRequired,
            totalReadyPods: PropTypes.number.isRequired,
            totalRegisteredNodes: PropTypes.number.isRequired,
        }),
        healthInfoComplete: PropTypes.bool,
        overallHealthStatus: PropTypes.string,
        sensorHealthStatus: PropTypes.string,
        lastContact: PropTypes.string, // ISO 8601
    }),
    status: PropTypes.shape({
        certExpiryStatus: PropTypes.shape({
            sensorCertExpiry: PropTypes.string,
        }),
        sensorVersion: PropTypes.string,
        upgradeStatus: PropTypes.shape({
            upgradability: PropTypes.string,
            mostRecentProcess: PropTypes.shape({
                active: PropTypes.bool,
                progress: PropTypes.shape({
                    upgradeState: PropTypes.string,
                    upgradeStatusDetail: PropTypes.string,
                }),
                type: PropTypes.string,
            }),
        }),
    }),
    centralVersion: PropTypes.string.isRequired,
    currentDatetime: PropTypes.instanceOf(Date).isRequired,
};

ClusterHealth.defaultProps = {
    healthStatus: null,
    status: null,
};

export default ClusterHealth;
