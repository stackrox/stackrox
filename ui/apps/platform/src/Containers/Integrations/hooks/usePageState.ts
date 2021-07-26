import { useLocation, useRouteMatch } from 'react-router-dom';

import { integrationCreatePath, integrationDetailsPath, integrationEditPath } from 'routePaths';
import { IntegrationSource, IntegrationType } from '../utils/integrationUtils';

type Params = {
    source: IntegrationSource;
    type: IntegrationType;
    id?: string;
};

type Location = { pathname: string };

type Match = { isExact: boolean; params: Params };

export type PageStates = 'CREATE' | 'EDIT' | 'VIEW_DETAILS';

type UsePageStateResult = {
    pageState: PageStates;
    params: {
        source: IntegrationSource;
        type: IntegrationType;
        id?: string;
    };
    isCreating: boolean;
    isEditing: boolean;
    isViewingDetails: boolean;
};

function usePageState(): UsePageStateResult {
    const location: Location = useLocation();
    const matchCreate: Match = useRouteMatch(integrationCreatePath);
    const matchEdit: Match = useRouteMatch(integrationEditPath);
    const matchViewDetails: Match = useRouteMatch(integrationDetailsPath);

    if (matchCreate?.isExact) {
        return {
            pageState: 'CREATE',
            params: matchCreate.params,
            isCreating: true,
            isEditing: false,
            isViewingDetails: false,
        };
    }
    if (matchEdit?.isExact) {
        return {
            pageState: 'EDIT',
            params: matchEdit.params,
            isCreating: false,
            isEditing: true,
            isViewingDetails: false,
        };
    }
    if (matchViewDetails?.isExact) {
        return {
            pageState: 'VIEW_DETAILS',
            params: matchViewDetails.params,
            isCreating: false,
            isEditing: false,
            isViewingDetails: true,
        };
    }
    throw new Error(`No valid page state exists for the current URL path (${location.pathname})`);
}

export default usePageState;
