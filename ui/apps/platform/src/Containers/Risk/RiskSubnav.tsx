import { NavList } from '@patternfly/react-core';

import { riskFullViewPath, riskPlatformViewPath, riskUserWorkloadsViewPath } from 'routePaths';
import useFilteredWorkflowViewURLState from 'Components/FilteredWorkflowViewSelector/useFilteredWorkflowViewURLState';
import {
    fullWorkflowView,
    platformWorkflowView,
    userWorkloadWorkflowView,
} from 'Components/FilteredWorkflowViewSelector/types';
import NavigationItem from 'Containers/MainPage/Navigation/NavigationItem';

function RiskSubnav() {
    const { filteredWorkflowView } = useFilteredWorkflowViewURLState();

    const isUserWorkloadsActive = filteredWorkflowView === userWorkloadWorkflowView;
    const isPlatformActive = filteredWorkflowView === platformWorkflowView;
    const isFullViewActive = filteredWorkflowView === fullWorkflowView;

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
