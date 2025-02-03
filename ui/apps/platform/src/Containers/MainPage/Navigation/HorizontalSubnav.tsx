import React, { useState } from 'react';
import { matchPath, useHistory, useLocation } from 'react-router-dom';
import {
    Nav,
    Dropdown,
    DropdownItem,
    DropdownList,
    MenuToggle,
    MenuToggleElement,
    NavItemSeparator,
    NavItem,
    NavList,
} from '@patternfly/react-core';

import {
    vulnerabilitiesNodeCvesPath,
    vulnerabilitiesUserWorkloadsPath,
    vulnerabilitiesPlatformPath,
    vulnerabilitiesAllImagesPath,
    vulnerabilitiesInactiveImagesPath,
} from 'routePaths';
import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { HasReadAccess } from 'hooks/usePermissions';
import { ensureExhaustive } from 'utils/type.utils';
import NavigationItem from './NavigationItem';
import { filterNavDescriptions, isActiveLink, NavDescription } from './utils';

import './HorizontalSubnav.css';

type SubnavParentKey = 'vulnerabilities';

/*
 * Function that returns a key/value object that maps parent routes to a list
 * of sub-navigation description items.
 */
function getSubnavDescriptionGroups(
    isFeatureFlagEnabled: IsFeatureFlagEnabled
): Record<SubnavParentKey, NavDescription[]> {
    return {
        vulnerabilities: isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT')
            ? [
                  {
                      type: 'link',
                      content: 'User Workloads',
                      path: vulnerabilitiesUserWorkloadsPath,
                      routeKey: 'vulnerabilities/user-workloads',
                  },
                  {
                      type: 'link',
                      content: 'Platform',
                      path: vulnerabilitiesPlatformPath,
                      routeKey: 'vulnerabilities/platform',
                  },
                  {
                      type: 'link',
                      content: 'Nodes',
                      path: vulnerabilitiesNodeCvesPath,
                      routeKey: 'vulnerabilities/node-cves',
                  },
                  {
                      type: 'parent',
                      key: 'More Views',
                      title: 'More Views',
                      children: [
                          {
                              type: 'link',
                              content: 'All Images',
                              description:
                                  'View findings for user and platform images simultaneously',
                              path: vulnerabilitiesAllImagesPath,
                              routeKey: 'vulnerabilities/all-images',
                          },
                          {
                              type: 'link',
                              content: 'Inactive images',
                              description:
                                  'View findings for images not currently deployed as workloads',
                              path: vulnerabilitiesInactiveImagesPath,
                              routeKey: 'vulnerabilities/inactive-images',
                          },
                      ],
                  },
              ]
            : [],
    };
}

/*
 * Given the mapping of parent routes to subnav description groups, return the grouping
 * that contains a child item with a path matching the user's current path
 */
function getSubnavGroupForCurrentPath(
    subnavDescriptionGroups: Record<SubnavParentKey, NavDescription[]>,
    pathname: string
) {
    return (
        Object.values(subnavDescriptionGroups).find((subnavDescriptionGroup) => {
            return subnavDescriptionGroup.some((group) => {
                if (group.type === 'link') {
                    return matchPath(pathname, group.path);
                }
                return group.children
                    .filter((child) => child.type === 'link')
                    .some(({ path }) => matchPath(pathname, path));
            });
        }) ?? []
    );
}

export type HorizontalSubnavProps = {
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function HorizontalSubnav({ hasReadAccess, isFeatureFlagEnabled }: HorizontalSubnavProps) {
    const history = useHistory();
    const { pathname } = useLocation();
    const routePredicates = { hasReadAccess, isFeatureFlagEnabled };

    const subnavDescriptionGroups = getSubnavDescriptionGroups(isFeatureFlagEnabled);
    const subnavDescriptionGroupForCurrentPath = getSubnavGroupForCurrentPath(
        subnavDescriptionGroups,
        pathname
    );
    const subnavDescriptions = filterNavDescriptions(
        subnavDescriptionGroupForCurrentPath,
        routePredicates
    );

    const [openDropdownKey, setOpenDropdownKey] = useState<string | null>(null);

    const onToggleClick = (key: string) => {
        setOpenDropdownKey((currentKey) => (currentKey === key ? null : key));
    };

    const onSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        history.push(value);
        setOpenDropdownKey(null);
    };

    if (!subnavDescriptions.length) {
        return null;
    }

    return (
        <Nav variant="horizontal-subnav" className="acs-pf-horizontal-subnav">
            <NavList>
                {subnavDescriptions.map((subnavDescription) => {
                    switch (subnavDescription.type) {
                        case 'link': {
                            const { content, path } = subnavDescription;
                            return (
                                <NavigationItem
                                    key={path}
                                    isActive={isActiveLink(pathname, subnavDescription)}
                                    path={path}
                                    content={
                                        typeof content === 'function'
                                            ? content(subnavDescriptions)
                                            : content
                                    }
                                />
                            );
                        }
                        case 'parent': {
                            const { key, title, children } = subnavDescription;
                            return (
                                <Dropdown
                                    key={key}
                                    isPlain
                                    onSelect={onSelect}
                                    isOpen={openDropdownKey === key}
                                    onOpenChange={(isOpen: boolean) =>
                                        setOpenDropdownKey(isOpen ? key : null)
                                    }
                                    toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                                        <NavItem
                                            isActive={subnavDescription.children.some(
                                                (child) =>
                                                    child.type === 'link' &&
                                                    isActiveLink(pathname, child)
                                            )}
                                            onClick={() => onToggleClick(key)}
                                        >
                                            <MenuToggle
                                                ref={toggleRef}
                                                isExpanded={openDropdownKey === key}
                                                variant="plainText"
                                            >
                                                {typeof title === 'function'
                                                    ? title(subnavDescriptions)
                                                    : title}
                                            </MenuToggle>
                                        </NavItem>
                                    )}
                                    shouldFocusToggleOnSelect
                                >
                                    <DropdownList>
                                        {children.map((child) => {
                                            if (child.type === 'separator') {
                                                return (
                                                    <NavItemSeparator
                                                        key={child.key}
                                                        role="listitem"
                                                    />
                                                );
                                            }
                                            const { content, path, description } = child;
                                            return (
                                                <DropdownItem
                                                    component={'a'}
                                                    className={
                                                        isActiveLink(pathname, child)
                                                            ? 'acs-pf-horizontal-subnav-menu__active'
                                                            : ''
                                                    }
                                                    value={path}
                                                    key={path}
                                                    description={description}
                                                >
                                                    {typeof content === 'function'
                                                        ? content(subnavDescriptions)
                                                        : content}
                                                </DropdownItem>
                                            );
                                        })}
                                    </DropdownList>
                                </Dropdown>
                            );
                        }
                        default:
                            return ensureExhaustive(subnavDescription);
                    }
                })}
            </NavList>
        </Nav>
    );
}

export default HorizontalSubnav;
