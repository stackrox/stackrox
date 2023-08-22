import { isRouteEnabled } from 'routePaths';

import useFeatureFlags from './useFeatureFlags';
import usePermissions from './usePermissions';

function useIsRouteEnabled(): (path: string) => boolean {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const { hasReadAccess } = usePermissions();

    return (path: string) => isRouteEnabled({ hasReadAccess, isFeatureFlagEnabled }, path);
}

export default useIsRouteEnabled;
