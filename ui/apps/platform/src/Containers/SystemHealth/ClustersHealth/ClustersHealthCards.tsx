import React, { ReactElement, useEffect, useState } from 'react';
import { GridItem } from '@patternfly/react-core';

import { fetchClustersAsArray } from 'services/ClustersService';
import { Cluster } from 'types/cluster.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ClusterStatusCard from './ClusterStatusCard';
import CredentialExpirationCard from './CredentialExpirationCard';
import SensorUpgradeCard from './SensorUpgradeCard';

type ClustersHealthCardsProps = {
    pollingCount: number;
};

const ClustersHealthCards = ({ pollingCount }: ClustersHealthCardsProps): ReactElement => {
    const [isFetching, setIsFetching] = useState(false);
    const [requestErrorMessage, setRequestErrorMessage] = useState('');
    const [clusters, setClusters] = useState<Cluster[]>([]);

    const [currentDatetime, setCurrentDatetime] = useState<Date | null>(null);

    useEffect(() => {
        setIsFetching(true);
        fetchClustersAsArray()
            .then((clustersFetched) => {
                setRequestErrorMessage('');
                // TODO supersede src/Containers/Clusters/clusterTypes.ts with types/cluster.proto.ts
                // eslint-disable-next-line
                // @ts-ignore
                setClusters(clustersFetched);
                setCurrentDatetime(new Date());
            })
            .catch((error) => {
                setRequestErrorMessage(getAxiosErrorMessage(error));
                setClusters([]);
                setCurrentDatetime(null);
            })
            .finally(() => {
                setIsFetching(false);
            });
    }, [pollingCount]);

    const isFetching0 = isFetching && pollingCount === 0;

    return (
        <>
            <GridItem span={12}>
                <ClusterStatusCard
                    clusters={clusters}
                    isFetching0={isFetching0}
                    requestErrorMessage={requestErrorMessage}
                />
            </GridItem>
            <GridItem span={12}>
                <SensorUpgradeCard
                    clusters={clusters}
                    isFetching0={isFetching0}
                    requestErrorMessage={requestErrorMessage}
                />
            </GridItem>
            <GridItem span={12}>
                <CredentialExpirationCard
                    clusters={clusters}
                    currentDatetime={currentDatetime}
                    isFetching0={isFetching0}
                    requestErrorMessage={requestErrorMessage}
                />
            </GridItem>
        </>
    );
};

export default ClustersHealthCards;
