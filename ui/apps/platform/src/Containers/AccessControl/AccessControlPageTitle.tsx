import React, { ReactElement } from 'react';
import pluralize from 'pluralize';

import PageTitle from 'Components/PageTitle';
import { AccessControlEntityType } from 'constants/entityTypes';
import useCaseTypes from 'constants/useCaseTypes';
import { accessControlLabels } from 'messages/common';
import useCaseLabels from 'messages/useCase';

const accessControlLabel = useCaseLabels[useCaseTypes.ACCESS_CONTROL];

export type AccessControlPageTitleProps = {
    entityType: AccessControlEntityType;
    isEntity: boolean;
};

function AccessControlPageTitle({
    entityType,
    isEntity,
}: AccessControlPageTitleProps): ReactElement {
    const entityLabel = accessControlLabels[entityType];
    const title = `${accessControlLabel} - ${isEntity ? entityLabel : pluralize(entityLabel)}`;

    return <PageTitle title={title} />;
}

export default AccessControlPageTitle;
