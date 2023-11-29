import React, { ReactElement, useEffect, useState } from 'react';
import { GridItem } from '@patternfly/react-core';

import { fetchClusters } from 'services/ClustersService';
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
    const [errorMessageFetching, setErrorMessageFetching] = useState('');
    const [clusters, setClusters] = useState<Cluster[]>([]);

    const [currentDatetime, setCurrentDatetime] = useState<Date | null>(null);

    useEffect(() => {
        setIsFetching(true);
        fetchClusters()
            .then((clustersFetched) => {
                setErrorMessageFetching('');
                // TODO supersede src/Containers/Clusters/clusterTypes.ts with types/cluster.proto.ts
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                setClusters(clustersFetched);
                setCurrentDatetime(new Date());
            })
            .catch((error) => {
                setErrorMessageFetching(getAxiosErrorMessage(error));
                setClusters([]);
                setCurrentDatetime(null);
            })
            .finally(() => {
                setIsFetching(false);
            });
    }, [pollingCount]);

    const isFetchingInitialRequest = isFetching && pollingCount === 0;

    return (
        <>
            <GridItem span={12}>
                <ClusterStatusCard
                    clusters={clusters}
                    isFetchingInitialRequest={isFetchingInitialRequest}
                    errorMessageFetching={errorMessageFetching}
                />
            </GridItem>
            <GridItem span={12}>
                <SensorUpgradeCard
                    clusters={clusters}
                    isFetchingInitialRequest={isFetchingInitialRequest}
                    errorMessageFetching={errorMessageFetching}
                />
            </GridItem>
            <GridItem span={12}>
                <CredentialExpirationCard
                    clusters={clusters}
                    currentDatetime={currentDatetime}
                    isFetchingInitialRequest={isFetchingInitialRequest}
                    errorMessageFetching={errorMessageFetching}
                />
            </GridItem>
        </>
    );
};

export default ClustersHealthCards;
