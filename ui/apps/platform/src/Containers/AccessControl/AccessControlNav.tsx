import React, { ReactElement } from 'react';
import pluralize from 'pluralize';

import TabNav from 'Components/TabNav';
import { AccessControlEntityType } from 'constants/entityTypes';
import { accessControlLabels } from 'messages/common';

import { entityPathSegment, getEntityPath } from './accessControlPaths';

export type AccessControlNavProps = {
    entityType?: AccessControlEntityType;
    isDisabled?: boolean;
};

/*
 * Render Access Control nav with PatternFly classes to make the following changes:
 * Render text instead of link if disabled while creating or editing.
 * Omit left and right scroll buttons.
 */
function AccessControlNav({ entityType, isDisabled }: AccessControlNavProps): ReactElement {
    const tabLinks = Object.keys(entityPathSegment).map((itemType) => {
        return {
            href: getEntityPath(itemType as AccessControlEntityType),
            title: pluralize(accessControlLabels[itemType]),
        };
    });
    return (
        <TabNav
            tabLinks={tabLinks}
            currentTabTitle={pluralize(accessControlLabels[entityType || 0])}
            isDisabled={isDisabled}
        />
    );
}

export default AccessControlNav;
