import React from 'react';
import { useLocation, Location } from 'react-router-dom';
import { networkBasePath, networkBasePathPF } from 'routePaths';

import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';

import LeftNavItem from './LeftNavItem';

type NetworkGraphNavItemsProps = {
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function NetworkGraphNavItems({ isFeatureFlagEnabled }: NetworkGraphNavItemsProps) {
    const location: Location = useLocation();
    const isNetworkGraphPFEnabled = isFeatureFlagEnabled('ROX_NETWORK_GRAPH_PATTERNFLY');

    const networkGraphTitle = isNetworkGraphPFEnabled
        ? 'Network Graph (1.0 deprecated)'
        : 'Network Graph';
    const networkGraphPFTitle = 'Network Graph (2.0)';

    return (
        <>
            {isNetworkGraphPFEnabled && (
                <LeftNavItem
                    isActive={location.pathname.includes(networkBasePathPF)}
                    path={networkBasePathPF}
                    title={networkGraphPFTitle}
                />
            )}

            <LeftNavItem
                isActive={
                    location.pathname.includes(networkBasePath) &&
                    !location.pathname.includes(networkBasePathPF)
                }
                path={networkBasePath}
                title={networkGraphTitle}
            />
        </>
    );
}

export default NetworkGraphNavItems;
