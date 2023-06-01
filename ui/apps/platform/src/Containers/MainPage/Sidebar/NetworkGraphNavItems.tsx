import React from 'react';
import { useLocation, Location } from 'react-router-dom';
import { networkBasePath, networkBasePathPF } from 'routePaths';

import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';

import BadgedNavItem from './BadgedNavItem';

type NetworkGraphNavItemsProps = {
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function NetworkGraphNavItems({ isFeatureFlagEnabled }: NetworkGraphNavItemsProps) {
    const location: Location = useLocation();
    const isNetworkGraphPFEnabled = isFeatureFlagEnabled('ROX_NETWORK_GRAPH_PATTERNFLY');

    const networkGraphTitle = isNetworkGraphPFEnabled ? 'Network Graph (1.0)' : 'Network Graph';
    const networkGraphPFTitle = 'Network Graph (2.0)';

    return (
        <>
            {isNetworkGraphPFEnabled && (
                <BadgedNavItem
                    variant="TechPreview"
                    isActive={location.pathname.includes(networkBasePathPF)}
                    path={networkBasePathPF}
                    title={networkGraphPFTitle}
                />
            )}

            <BadgedNavItem
                variant="Deprecated"
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
