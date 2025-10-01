import React from 'react';
import type { ReactElement } from 'react';
import pluralize from 'pluralize';

import PageTitle from 'Components/PageTitle';
import type { AccessControlEntityType } from 'constants/entityTypes';
import useCaseTypes from 'constants/useCaseTypes';
import { accessControlLabels } from 'messages/common';
import useCaseLabels from 'messages/useCase';

const accessControlLabel = useCaseLabels[useCaseTypes.ACCESS_CONTROL];

export type AccessControlPageTitleProps = {
    entityType: AccessControlEntityType;
    isList: boolean;
};

function AccessControlPageTitle({ entityType, isList }: AccessControlPageTitleProps): ReactElement {
    const entityLabel = accessControlLabels[entityType];
    const title = `${accessControlLabel} - ${isList ? pluralize(entityLabel) : entityLabel}`;

    return <PageTitle title={title} />;
}

export default AccessControlPageTitle;
