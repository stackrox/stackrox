import { useLocation } from 'react-router-dom-v5-compat';
import { NavList } from '@patternfly/react-core';

import {
    violationsFullViewPath,
    violationsPlatformViewPath,
    violationsUserWorkloadsViewPath,
} from 'routePaths';
import { hasSearchKeyValue } from 'utils/searchUtils';
import NavigationItem from 'Containers/MainPage/Navigation/NavigationItem';
import type { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import type { HasReadAccess } from 'hooks/usePermissions';

type ViolationsSubnavProps = {
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function ViolationsSubnav({
    hasReadAccess, // eslint-disable-line @typescript-eslint/no-unused-vars
    isFeatureFlagEnabled, // eslint-disable-line @typescript-eslint/no-unused-vars
}: ViolationsSubnavProps) {
    const location = useLocation();
    const search = location.search || '';

    const isUserWorkloadsActive =
        hasSearchKeyValue(search, 'filteredWorkflowView', 'Applications view') ||
        hasSearchKeyValue(search, 'filteredWorkflowView', null);
    const isPlatformActive = hasSearchKeyValue(search, 'filteredWorkflowView', 'Platform view');
    const isFullViewActive = hasSearchKeyValue(search, 'filteredWorkflowView', 'Full view');

    return (
        <NavList>
            <NavigationItem
                isActive={isUserWorkloadsActive}
                path={violationsUserWorkloadsViewPath}
                content="User Workloads"
            />
            <NavigationItem
                isActive={isPlatformActive}
                path={violationsPlatformViewPath}
                content="Platform"
            />
            <NavigationItem
                isActive={isFullViewActive}
                path={violationsFullViewPath}
                content="All Violations"
            />
        </NavList>
    );
}

export default ViolationsSubnav;
