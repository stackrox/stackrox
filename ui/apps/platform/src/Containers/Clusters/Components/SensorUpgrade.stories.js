import React from 'react';

import SensorUpgrade from './SensorUpgrade';

export default {
    title: 'SensorUpgrade',
    component: SensorUpgrade,
};

const clusterId = '12345678-1234-1234-1234-1234567890ab';

const upgradeSingleCluster = (clusterArg) => {
    // eslint-disable-next-line no-alert
    alert(`upgradeSingleCluster(${clusterArg})`);
};

// Provide realistic inherited styles.
const bodyClassName = 'font-sans text-base-600 text-base font-600';
const heading = 'Sensor Upgrade';

const AtSide = ({ children }) => (
    <div className={bodyClassName}>
        <table style={{ width: '20rem' }}>
            <tbody>
                <tr className="align-top leading-normal">
                    <th className="pl-0 pr-2 py-1 text-left whitespace-no-wrap" scope="row">
                        {heading}
                    </th>
                    <td className="px-0 py-1">{children}</td>
                </tr>
            </tbody>
        </table>
    </div>
);

const InList = ({ children }) => (
    <div className={bodyClassName}>
        <div className="ReactTable" style={{ fontSize: '0.75rem', width: '18rem' }}>
            <div className="rt-table" role="grid">
                <div className="rt-thead pl-3">
                    <div className="rt-tr" role="row">
                        <div className="rt-th px-2 py-4 pb-3 font-700 text-left">
                            <div>{heading}</div>
                        </div>
                    </div>
                </div>
                <div className="rt-tbody">
                    <div className="rt-tr" role="row">
                        <div className="rt-td p-2 flex items-center text-left">{children}</div>
                    </div>
                </div>
            </div>
        </div>
    </div>
);

export const typeCompleteWithoutMostRecentProcessAtSide = () => (
    <AtSide>
        <SensorUpgrade
            upgradeStatus={{ upgradability: 'UP_TO_DATE' }}
            sensorVersion="3.47.0.0"
            centralVersion="3.47.0.0"
            isList={false}
            actionProps={null}
        />
    </AtSide>
);

export const typeCompleteWithMostRecentProcessInList = () => (
    <InList>
        <SensorUpgrade
            upgradeStatus={{
                upgradability: 'UP_TO_DATE',
                mostRecentProcess: {
                    active: false,
                    progress: { upgradeState: 'UPGRADE_COMPLETE' },
                },
            }}
            sensorVersion="3.47.0.0"
            centralVersion="3.47.0.0"
            isList
            actionProps={{
                clusterId,
                upgradeSingleCluster,
            }}
        />
    </InList>
);

export const typeDownloadAtSide = () => (
    <AtSide>
        <SensorUpgrade
            upgradeStatus={{ upgradability: 'AUTO_UPGRADE_POSSIBLE' }}
            sensorVersion="3.46.0.0"
            centralVersion="3.47.0.0"
            isList={false}
            actionProps={null}
        />
    </AtSide>
);

export const typeDownloadInList = () => (
    <InList>
        <SensorUpgrade
            upgradeStatus={{ upgradability: 'AUTO_UPGRADE_POSSIBLE' }}
            sensorVersion="3.46.0.0"
            centralVersion="3.47.0.0"
            isList
            actionProps={{
                clusterId,
                upgradeSingleCluster,
            }}
        />
    </InList>
);

export const typeInterventionAtSide = () => (
    <AtSide>
        <SensorUpgrade
            upgradeStatus={{ upgradability: 'MANUAL_UPGRADE_REQUIRED' }}
            sensorVersion="3.46.0.0"
            centralVersion="3.47.0.0"
            isList={false}
            actionProps={null}
        />
    </AtSide>
);

export const typeProgressInList = () => (
    <InList>
        <SensorUpgrade
            upgradeStatus={{
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: { upgradeState: 'UPGRADER_LAUNCHING' },
                },
            }}
            sensorVersion="3.46.0.0"
            centralVersion="3.47.0.0"
            isList
            actionProps={{
                clusterId,
                upgradeSingleCluster,
            }}
        />
    </InList>
);

export const typeFailureWithoutActionInList = () => (
    <InList>
        <SensorUpgrade
            upgradeStatus={{
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: { upgradeState: 'UPGRADE_ERROR_ROLLING_BACK' },
                },
            }}
            sensorVersion="3.46.0.0"
            centralVersion="3.47.0.0"
            isList
            actionProps={{
                clusterId,
                upgradeSingleCluster,
            }}
        />
    </InList>
);

export const typeFailureWithActionInList = () => (
    <InList>
        <SensorUpgrade
            upgradeStatus={{
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: false,
                    progress: { upgradeState: 'UPGRADE_ERROR_ROLLED_BACK' },
                },
            }}
            sensorVersion="3.46.0.0"
            centralVersion="3.47.0.0"
            isList
            actionProps={{
                clusterId,
                upgradeSingleCluster,
            }}
        />
    </InList>
);

export const typeFailureAtSide = () => (
    <AtSide>
        <SensorUpgrade
            upgradeStatus={{
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: false,
                    progress: { upgradeState: 'UPGRADE_ERROR_ROLLED_BACK' },
                },
            }}
            sensorVersion="3.46.0.0"
            centralVersion="3.47.0.0"
            isList={false}
            actionProps={null}
        />
    </AtSide>
);
