import { RouteKey, isRouteEnabled } from 'routePaths';

import useFeatureFlags from './useFeatureFlags';
import usePermissions from './usePermissions';

export type IsRouteEnabled = (routeKey: RouteKey) => boolean;

function useIsRouteEnabled(): IsRouteEnabled {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const { hasReadAccess } = usePermissions();

    return (routeKey: RouteKey) =>
        isRouteEnabled({ hasReadAccess, isFeatureFlagEnabled }, routeKey);
}

export default useIsRouteEnabled;
