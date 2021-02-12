import React, { ReactElement } from 'react';
import pluralize from 'pluralize';

import PageHeader from 'Components/PageHeader';
import TileLink from 'Components/TileLink';
import { AccessControlEntityType } from 'constants/entityTypes';
import { accessControlLabels } from 'messages/common';

import { entityPathSegment, getEntityPath } from './accessControlPaths';

const linkTypes: AccessControlEntityType[] = [
    'AUTH_PROVIDER',
    'ROLE',
    'PERMISSION_SET',
    'ACCESS_SCOPE',
];

export type AccessControlPageHeaderProps = {
    currentType: AccessControlEntityType;
};

/*
 * Render links to list pages with visual distinction for the current route.
 */
function AccessControlPageHeader({ currentType }: AccessControlPageHeaderProps): ReactElement {
    // TODO replace TileLink with future component
    // TODO factor out divs as presentation component? after replacing component
    return (
        <PageHeader
            header="Access Control"
            subHeader="Configure authentication, permissions, and scope"
            classes="pr-0 ignore-react-onclickoutside"
        >
            <div className="flex flex-1 h-full items-center justify-end">
                <div className="flex h-10 pr-2">
                    {linkTypes.map((linkType) => (
                        <TileLink
                            key={linkType}
                            text={pluralize(accessControlLabels[linkType])}
                            url={getEntityPath(linkType)}
                            colorClasses={linkType === currentType ? 'border-base-600' : ''}
                            short
                            dataTestId={`${entityPathSegment[linkType]}-link`}
                        />
                    ))}
                </div>
            </div>
        </PageHeader>
    );
}

export default AccessControlPageHeader;
