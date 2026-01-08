import { useLocation } from 'react-router-dom-v5-compat';
import { NavList } from '@patternfly/react-core';

import {
    violationsFullViewPath,
    violationsPlatformViewPath,
    violationsUserWorkloadsViewPath,
} from 'routePaths';
import { hasSearchKeyValue } from 'utils/searchUtils';
import NavigationItem from 'Components/Navigation/NavigationItem';

function ViolationsSubnav() {
    const location = useLocation();
    const search = location.search || '';

    const isUserWorkloadsActive =
        hasSearchKeyValue(search, 'filteredWorkflowView', 'Applications view') ||
        hasSearchKeyValue(search, 'filteredWorkflowView', null);
    const isPlatformActive = hasSearchKeyValue(search, 'filteredWorkflowView', 'Platform view');
    const isAllViolationsActive = hasSearchKeyValue(search, 'filteredWorkflowView', 'Full view');

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
                isActive={isAllViolationsActive}
                path={violationsFullViewPath}
                content="All Violations"
            />
        </NavList>
    );
}

export default ViolationsSubnav;
