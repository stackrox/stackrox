import { ReactElement } from 'react';
import { Location, matchPath } from 'react-router-dom';

import { isRouteEnabled, RouteKey } from 'routePaths';
import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { HasReadAccess } from 'hooks/usePermissions';

// Child example; Compliance (1.0) if Compliance (2.0) is rendered and Compliance otherwise.
// Parent example: Vulnerability Management (1.0) if Vulnerability Management (2.0) is rendered and so on.
type TitleCallback = (navDescriptionFiltered: NavDescription[]) => string | ReactElement;

type IsActiveCallback = (location: Location) => boolean;

export type LinkDescription = {
    type: 'link';
    content: string | TitleCallback | ReactElement;
    path: string;
    routeKey: RouteKey;
    description?: string;
    isActive?: IsActiveCallback; // for example, exact match
};

// Encapsulate whether path match for child is specific or generic.
export function isActiveLink(location: Location, { isActive, path }: LinkDescription) {
    return typeof isActive === 'function'
        ? isActive(location)
        : Boolean(matchPath({ path }, location.pathname));
}

export type SeparatorDescription = {
    type: 'separator';
    key: string; // corresponds to React key prop
};

export type ChildDescription = LinkDescription | SeparatorDescription;

export type ParentDescription = {
    type: 'parent';
    title: string | ReactElement | TitleCallback;
    key: string; // for key prop and especially for title callback
    children: ChildDescription[];
};

export type NavDescription = LinkDescription | ParentDescription;

export type RoutePredicates = {
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function isChildLinkEnabled(childDescription: ChildDescription, routePredicates: RoutePredicates) {
    return childDescription.type === 'link'
        ? isRouteEnabled(routePredicates, childDescription.routeKey)
        : true;
}

function isChildSeparatorRelevant(
    childDescription: ChildDescription,
    index: number,
    array: ChildDescription[]
) {
    // A separator is relevant if it is preceded and followed by a link whose route is enabled.
    return childDescription.type === 'separator'
        ? index !== 0 && index !== array.length - 1 && array[index + 1].type === 'link'
        : true;
}

export function filterNavDescriptions<T extends NavDescription>(
    navDescriptions: T[],
    routePredicates: RoutePredicates
) {
    return navDescriptions
        .map((navDescription) => {
            switch (navDescription.type) {
                case 'parent': {
                    // Filter second-level children.
                    return {
                        ...navDescription,
                        children: navDescription.children
                            .filter((child) => isChildLinkEnabled(child, routePredicates))
                            .filter(isChildSeparatorRelevant),
                    };
                }
                default: {
                    return navDescription;
                }
            }
        })
        .filter((navDescription) => {
            // Filter first-level parents and children.
            switch (navDescription.type) {
                case 'parent': {
                    return navDescription.children.length !== 0;
                }
                default: {
                    return isRouteEnabled(routePredicates, navDescription.routeKey);
                }
            }
        });
}
