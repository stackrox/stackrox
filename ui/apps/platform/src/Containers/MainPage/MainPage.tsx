import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import { useSelector } from 'react-redux';
import { useNavigate } from 'react-router-dom-v5-compat';

import LoadingSection from 'Components/PatternFly/LoadingSection';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
import usePublicConfig from 'hooks/usePublicConfig';
import { selectors } from 'reducers';
import { getClustersForPermissions } from 'services/RolesService';
import { clustersBasePath } from 'routePaths';

import { CommandCenterShell } from './CommandCenterShell';

function MainPage(): ReactElement {
    const navigate = useNavigate();

    const { isFeatureFlagEnabled, isLoadingFeatureFlags } = useFeatureFlags();
    const { hasReadAccess, hasReadWriteAccess, isLoadingPermissions } = usePermissions();
    const { publicConfig, isLoadingPublicConfig } = usePublicConfig();
    const isLoadingCentralCapabilities = useSelector(selectors.getIsLoadingCentralCapabilities);
    const [isLoadingClustersCount, setIsLoadingClustersCount] = useState(false);

    const hasWriteAccessForCluster = hasReadWriteAccess('Cluster');

    useEffect(() => {
        if (hasWriteAccessForCluster) {
            setIsLoadingClustersCount(true);
            getClustersForPermissions([])
                .then(({ clusters }) => {
                    if (clusters?.length === 0) {
                        navigate(clustersBasePath);
                    }
                })
                .catch(() => {})
                .finally(() => {
                    setIsLoadingClustersCount(false);
                });
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [hasWriteAccessForCluster]);

    if (
        isLoadingFeatureFlags ||
        isLoadingPermissions ||
        (isLoadingPublicConfig && !publicConfig) ||
        isLoadingCentralCapabilities ||
        isLoadingClustersCount
    ) {
        return <LoadingSection message="Loading..." />;
    }

    return (
        <CommandCenterShell
            hasReadAccess={hasReadAccess}
            isFeatureFlagEnabled={isFeatureFlagEnabled}
        />
    );
}

export default MainPage;
