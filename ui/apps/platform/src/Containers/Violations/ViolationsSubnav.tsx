import { NavList } from '@patternfly/react-core';

import {
    violationsFullViewPath,
    violationsNodeViewPath,
    violationsPlatformViewPath,
    violationsUserWorkloadsViewPath,
} from 'routePaths';
import useFilteredWorkflowViewURLState from 'Components/FilteredWorkflowViewSelector/useFilteredWorkflowViewURLState';
import {
    fullWorkflowView,
    nodeWorkflowView,
    platformWorkflowView,
    userWorkloadWorkflowView,
} from 'Components/FilteredWorkflowViewSelector/types';
import NavigationItem from 'Containers/MainPage/Navigation/NavigationItem';

function ViolationsSubnav() {
    const { filteredWorkflowView } = useFilteredWorkflowViewURLState();

    const isUserWorkloadsActive = filteredWorkflowView === userWorkloadWorkflowView;
    const isPlatformActive = filteredWorkflowView === platformWorkflowView;
    const isNodeActive = filteredWorkflowView === nodeWorkflowView;
    const isFullViewActive = filteredWorkflowView === fullWorkflowView;

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
            <NavigationItem isActive={isNodeActive} path={violationsNodeViewPath} content="Nodes" />
            <NavigationItem
                isActive={isFullViewActive}
                path={violationsFullViewPath}
                content="All Violations"
            />
        </NavList>
    );
}

export default ViolationsSubnav;
