import React, { ReactElement } from 'react';
import { Helmet } from 'react-helmet';
import { useLocation } from 'react-router-dom';
import capitalize from 'lodash/capitalize';

import { basePathToLabelMap } from 'routePaths';
import parseURL from 'utils/URLParser';
import { resourceLabels } from 'messages/common';
import useCaseLabels from 'messages/useCase';

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

const getPageTitleText = (location: Location): string => {
    const baseTitleText = 'StackRox';
    if (basePathToLabelMap[location.pathname]) {
        const topPageLabel = basePathToLabelMap[location.pathname];
        return `${topPageLabel} | ${baseTitleText}`;
    }
    const workflowState = parseURL(location);
    if (workflowState) {
        const workflowPageLabel = getTitleFromWorkflowState(workflowState);
        return `${workflowPageLabel} | ${baseTitleText}`;
    }
    return baseTitleText;
};

const PageTitle = (): ReactElement => {
    const location = useLocation();
    const titleText = getPageTitleText(location);
    return (
        <Helmet>
            <title>{titleText}</title>
        </Helmet>
    );
};

export default PageTitle;
