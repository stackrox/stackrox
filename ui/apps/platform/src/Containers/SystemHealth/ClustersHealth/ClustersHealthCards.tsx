import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import { GridItem } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useMetadata from 'hooks/useMetadata';
import { fetchClusters } from 'services/ClustersService';
import type { Cluster } from 'types/cluster.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ClusterStatusCard from './ClusterStatusCard';
import CredentialExpirationCard from './CredentialExpirationCard';
import SensorCompatibilityCard from './SensorCompatibilityCard';
import SensorUpgradeCard from './SensorUpgradeCard';

type ClustersHealthCardsProps = {
    pollingCount: number;
};

const ClustersHealthCards = ({ pollingCount }: ClustersHealthCardsProps): ReactElement => {
    const [isFetching, setIsFetching] = useState(false);
    const [errorMessageFetching, setErrorMessageFetching] = useState('');
    const [clusters, setClusters] = useState<Cluster[]>([]);

    const [currentDatetime, setCurrentDatetime] = useState<Date | null>(null);

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const metadata = useMetadata();
    const showCompatibilityStatus = isFeatureFlagEnabled('ROX_SENSOR_COMPATIBILITY_STATUS');
    const compatibleVersions = metadata?.compatibleSensorVersions ?? [];

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
                {showCompatibilityStatus ? (
                    <SensorCompatibilityCard
                        clusters={clusters}
                        compatibleVersions={compatibleVersions}
                        isFetchingInitialRequest={isFetchingInitialRequest}
                        errorMessageFetching={errorMessageFetching}
                    />
                ) : (
                    <SensorUpgradeCard
                        clusters={clusters}
                        isFetchingInitialRequest={isFetchingInitialRequest}
                        errorMessageFetching={errorMessageFetching}
                    />
                )}
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
