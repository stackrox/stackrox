/* eslint-disable no-nested-ternary */
import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import pluralize from 'pluralize';

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
    return (
        <nav className="pf-c-nav pf-m-tertiary">
            <ul className="pf-c-nav__list">
                {Object.keys(entityPathSegment).map((itemType) => {
                    const isCurrent = itemType === entityType;
                    const className = isCurrent ? 'pf-c-nav__link pf-m-current' : 'pf-c-nav__link';
                    const path = getEntityPath(itemType as AccessControlEntityType);
                    const text = pluralize(accessControlLabels[itemType]);

                    return (
                        <li key={itemType} className="pf-c-nav__item">
                            {isDisabled ? (
                                <span className={className}>{text}</span>
                            ) : isCurrent ? (
                                <Link to={path} className={className} aria-current="page">
                                    {text}
                                </Link>
                            ) : (
                                <Link to={path} className={className}>
                                    {text}
                                </Link>
                            )}
                        </li>
                    );
                })}
            </ul>
        </nav>
    );
}

export default AccessControlNav;
