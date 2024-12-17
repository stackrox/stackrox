import { useLocation, useMatch } from 'react-router-dom';
import {
    integrationCreatePath,
    integrationDetailsPath,
    integrationEditPath,
    integrationsListPath,
    integrationsPath,
} from 'routePaths';
import { IntegrationSource, IntegrationType } from '../utils/integrationUtils';

type Params = {
    source: IntegrationSource;
    type: IntegrationType;
    id?: string;
};

type Location = { pathname: string };

export type PageStates = 'LIST' | 'CREATE' | 'EDIT' | 'VIEW_DETAILS';

type UsePageStateResult = {
    pageState: PageStates;
    params: {
        source: IntegrationSource;
        type: IntegrationType;
        id?: string;
    };
    isList: boolean;
    isCreating: boolean;
    isEditing: boolean;
    isViewingDetails: boolean;
    getPathToCreate: (source: IntegrationSource, type: IntegrationType) => string;
    getPathToEdit: (source: IntegrationSource, type: IntegrationType, id: string) => string;
    getPathToViewDetails: (source: IntegrationSource, type: IntegrationType, id: string) => string;
};

function usePageState(): UsePageStateResult {
    const location: Location = useLocation();
    const matchList = useMatch(integrationsListPath);
    const matchCreate = useMatch(integrationCreatePath);
    const matchEdit = useMatch(integrationEditPath);
    const matchViewDetails = useMatch(integrationDetailsPath);

    function getPathToCreate(source: IntegrationSource, type: IntegrationType): string {
        return `${integrationsPath}/${source}/${type}/create`;
    }

    function getPathToEdit(source: IntegrationSource, type: IntegrationType, id: string): string {
        return `${integrationsPath}/${source}/${type}/edit/${id}`;
    }

    function getPathToViewDetails(
        source: IntegrationSource,
        type: IntegrationType,
        id: string
    ): string {
        return `${integrationsPath}/${source}/${type}/view/${id}`;
    }

    if (matchList) {
        return {
            pageState: 'LIST',
            params: matchList.params as Params,
            isList: true,
            isCreating: false,
            isEditing: false,
            isViewingDetails: false,
            getPathToCreate,
            getPathToEdit,
            getPathToViewDetails,
        };
    }
    if (matchCreate) {
        return {
            pageState: 'CREATE',
            params: matchCreate.params as Params,
            isList: false,
            isCreating: true,
            isEditing: false,
            isViewingDetails: false,
            getPathToCreate,
            getPathToEdit,
            getPathToViewDetails,
        };
    }
    if (matchEdit) {
        return {
            pageState: 'EDIT',
            params: matchEdit.params as Params,
            isList: false,
            isCreating: false,
            isEditing: true,
            isViewingDetails: false,
            getPathToCreate,
            getPathToEdit,
            getPathToViewDetails,
        };
    }
    if (matchViewDetails) {
        return {
            pageState: 'VIEW_DETAILS',
            params: matchViewDetails.params as Params,
            isList: false,
            isCreating: false,
            isEditing: false,
            isViewingDetails: true,
            getPathToCreate,
            getPathToEdit,
            getPathToViewDetails,
        };
    }
    throw new Error(`No valid page state exists for the current URL path (${location.pathname})`);
}

export default usePageState;
