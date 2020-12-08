import React, { ReactElement } from 'react';

import ClusterHealth from './ClusterHealth';

import {
    Cluster,
    clusterStatusHealthyKey,
    clusterStatusStyleMap,
    credentialExpirationLabelMap,
    getCredentialExpirationCountMap,
    problemText,
} from '../utils/clusters';
import { nbsp } from '../utils/health';

const healthySubtext = 'There are no credential expirations this month';

const healthyText = {
    plural: `clusters with valid${nbsp}credentials`,
    singular: `cluster with valid${nbsp}credentials`,
};

type Props = {
    clusters: Cluster[];
    currentDatetime: Date;
};

const CredentialExpiration = ({ clusters, currentDatetime }: Props): ReactElement => (
    <ClusterHealth
        countMap={getCredentialExpirationCountMap(clusters, currentDatetime)}
        healthyKey={clusterStatusHealthyKey}
        healthySubtext={healthySubtext}
        healthyText={healthyText}
        labelMap={credentialExpirationLabelMap}
        problemText={problemText}
        styleMap={clusterStatusStyleMap}
    />
);

export default CredentialExpiration;
