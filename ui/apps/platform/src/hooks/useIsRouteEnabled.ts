import { isRouteEnabled } from 'routePaths';

import useFeatureFlags from './useFeatureFlags';
import usePermissions from './usePermissions';

function useIsRouteEnabled(path: string) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const { hasReadAccess } = usePermissions();

    return isRouteEnabled({ hasReadAccess, isFeatureFlagEnabled }, path);
}

export default useIsRouteEnabled;
