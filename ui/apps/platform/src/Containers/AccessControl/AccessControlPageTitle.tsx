import type { ReactElement } from 'react';
import pluralize from 'pluralize';

import PageTitle from 'Components/PageTitle';
import type { AccessControlEntityType } from 'constants/entityTypes';
import { accessControlLabels } from 'messages/common';

const accessControlLabel = 'Access Control';

type AccessControlPageTitleProps = {
    entityType: AccessControlEntityType;
    isList: boolean;
};

function AccessControlPageTitle({ entityType, isList }: AccessControlPageTitleProps): ReactElement {
    const entityLabel = accessControlLabels[entityType];
    const title = `${accessControlLabel} - ${isList ? pluralize(entityLabel) : entityLabel}`;

    return <PageTitle title={title} />;
}

export default AccessControlPageTitle;
