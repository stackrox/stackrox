import React from 'react';

import CredentialInteraction from './CredentialInteraction';

export default {
    title: 'CredentialInteraction',
    component: CredentialInteraction,
};

// Provide realistic inherited styles.
const bodyClassName = 'font-sans text-base-600 text-base font-600';

const AtSide = ({ children }) => <div className={bodyClassName}>{children}</div>;

const currentDatetime = new Date('2020-08-31T13:01:00Z');

// epsilon-edison-5
export const upgradeDisabled = () => (
    <AtSide>
        <CredentialInteraction
            certExpiryStatus={{
                sensorCertExpiry: '2020-09-07T13:00:00Z',
            }}
            currentDatetime={currentDatetime}
            upgradeStatus={{
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: null,
            }}
            clusterId="f342ca31-9271-081c-c40d-4e4cd442707a"
        />
    </AtSide>
);

// No story for downloaded, because it depends on a state change from interaction.

// eta-7
export const upgradeEnabled = () => (
    <AtSide>
        <CredentialInteraction
            certExpiryStatus={{
                sensorCertExpiry: '2020-09-29T13:01:00Z',
            }}
            currentDatetime={currentDatetime}
            upgradeStatus={{
                upgradability: 'UP_TO_DATE',
                upgradabilityStatusReason: 'sensor is running the same version as Central',
                mostRecentProcess: null,
            }}
            clusterId="2a226ad1-9f7a-eb6d-b0e5-a0db4ad68161"
        />
    </AtSide>
);

// eta-7
export const upgraded = () => (
    <AtSide>
        <CredentialInteraction
            certExpiryStatus={{
                sensorCertExpiry: '2020-09-29T13:01:00Z',
            }}
            currentDatetime={currentDatetime}
            upgradeStatus={{
                upgradability: 'UP_TO_DATE',
                upgradabilityStatusReason: 'sensor is running the same version as Central',
                mostRecentProcess: {
                    type: 'CERT_ROTATION',
                    progress: {
                        upgradeState: 'UPGRADE_COMPLETE',
                    },
                    initiatedAt: '2020-08-31T13:00:00Z',
                },
            }}
            clusterId="2a226ad1-9f7a-eb6d-b0e5-a0db4ad68161"
        />
    </AtSide>
);
