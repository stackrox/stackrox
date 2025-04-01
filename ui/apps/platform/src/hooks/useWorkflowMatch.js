import { useLocation, matchPath } from 'react-router-dom';

import { workflowPaths, validPageEntityListTypes, validPageEntityTypes } from 'routePaths';

function useWorkflowMatch() {
    const location = useLocation();

    const entityMatch = matchPath({ path: workflowPaths.ENTITY }, location.pathname);

    const listMatch = matchPath({ path: workflowPaths.LIST }, location.pathname);

    const dashboardMatch = matchPath({ path: workflowPaths.DASHBOARD }, location.pathname);

    if (entityMatch && validPageEntityTypes.includes(entityMatch.params.pageEntityType)) {
        return entityMatch;
    }

    if (listMatch && validPageEntityListTypes.includes(listMatch.params.pageEntityListType)) {
        return listMatch;
    }

    if (dashboardMatch) {
        return dashboardMatch;
    }

    return null;
}

export default useWorkflowMatch;
