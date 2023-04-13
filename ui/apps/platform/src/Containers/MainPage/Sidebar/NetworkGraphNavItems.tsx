import React from 'react';
import { useLocation, Location } from 'react-router-dom';
import { networkBasePath, networkBasePathPF } from 'routePaths';
import { Badge, Flex, FlexItem } from '@patternfly/react-core';

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
    const networkGraphPFTitle = (
        <Flex>
            <FlexItem>Network Graph</FlexItem>
            <FlexItem>
                <Badge
                    style={{
                        backgroundColor: 'var(--pf-global--palette--cyan-400)',
                    }}
                >
                    2.0
                </Badge>
            </FlexItem>
        </Flex>
    );

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
