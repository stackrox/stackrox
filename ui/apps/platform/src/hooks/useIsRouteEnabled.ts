import { RouteKey, isRouteEnabled } from 'routePaths';

import useFeatureFlags from './useFeatureFlags';
import usePermissions from './usePermissions';

function useIsRouteEnabled() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const { hasReadAccess } = usePermissions();

    return (routeKey: RouteKey) =>
        isRouteEnabled({ hasReadAccess, isFeatureFlagEnabled }, routeKey);
}

export default useIsRouteEnabled;
