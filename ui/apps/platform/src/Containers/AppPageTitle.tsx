import React, { ReactElement } from 'react';
import { useLocation } from 'react-router-dom';
import capitalize from 'lodash/capitalize';

import { basePathToLabelMap } from 'routePaths';
import parseURL from 'utils/URLParser';
import { resourceLabels } from 'messages/common';
import useCaseLabels from 'messages/useCase';

import PageTitle from 'Components/PageTitle';

type Location = {
    pathname: string;
};

const getTitleFromWorkflowState = (workflowState): string => {
    const useCase = useCaseLabels[workflowState.getUseCase()];
    const baseEntityType = resourceLabels[workflowState.getBaseEntityType()];
    if (baseEntityType) {
        return `${useCase} - ${capitalize(baseEntityType)}`;
    }
    return useCase;
};

const getPageTitleText = (location: Location): string | null | undefined => {
    if (basePathToLabelMap[location.pathname]) {
        return basePathToLabelMap[location.pathname];
    }
    const workflowState = parseURL(location);
    const useCase = workflowState.getUseCase();
    if (workflowState && useCase) {
        const workflowPageLabel = getTitleFromWorkflowState(workflowState);
        return workflowPageLabel;
    }
    return null;
};

const AppPageTitle = (): ReactElement => {
    const location = useLocation();
    const title = getPageTitleText(location);
    return <PageTitle title={title} />;
};

export default AppPageTitle;
