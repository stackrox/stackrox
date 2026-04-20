import { useLocation } from 'react-router-dom-v5-compat';
import { NavList } from '@patternfly/react-core';

import { riskFullViewPath, riskPlatformViewPath, riskUserWorkloadsViewPath } from 'routePaths';
import { hasSearchKeyValue } from 'utils/searchUtils';
import NavigationItem from 'Containers/MainPage/Navigation/NavigationItem';

function RiskSubnav() {
    const location = useLocation();
    const search = location.search ?? '';

    const isUserWorkloadsActive =
        hasSearchKeyValue(search, 'filteredWorkflowView', 'Applications view') ||
        hasSearchKeyValue(search, 'filteredWorkflowView', null);
    const isPlatformActive = hasSearchKeyValue(search, 'filteredWorkflowView', 'Platform view');
    const isFullViewActive = hasSearchKeyValue(search, 'filteredWorkflowView', 'Full view');

    return (
        <NavList>
            <NavigationItem
                isActive={isUserWorkloadsActive}
                path={riskUserWorkloadsViewPath}
                content="User Workloads"
            />
            <NavigationItem
                isActive={isPlatformActive}
                path={riskPlatformViewPath}
                content="Platform"
            />
            <NavigationItem
                isActive={isFullViewActive}
                path={riskFullViewPath}
                content="All Deployments"
            />
        </NavList>
    );
}

export default RiskSubnav;
