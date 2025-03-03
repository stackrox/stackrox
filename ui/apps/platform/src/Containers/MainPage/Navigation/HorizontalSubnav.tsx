import React, { useState } from 'react';
import { matchPath, useLocation, useNavigate } from 'react-router-dom';
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
    vulnerabilitiesImagesWithoutCvesPath,
    violationsFullViewPath,
    violationsPlatformViewPath,
    violationsUserWorkloadsViewPath,
    vulnerabilitiesPlatformCvesPath,
} from 'routePaths';
import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { HasReadAccess } from 'hooks/usePermissions';
import { ensureExhaustive } from 'utils/type.utils';
import NavigationItem from './NavigationItem';
import { filterNavDescriptions, isActiveLink, NavDescription } from './utils';

import './HorizontalSubnav.css';

type SubnavParentKey = 'violations' | 'vulnerabilities';

/*
 * Function that returns a key/value object that maps parent routes to a list
 * of sub-navigation description items.
 */
function getSubnavDescriptionGroups(
    isFeatureFlagEnabled: IsFeatureFlagEnabled
): Record<SubnavParentKey, NavDescription[]> {
    return {
        violations: isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT')
            ? [
                  {
                      type: 'link',
                      content: 'User Workloads',
                      path: violationsUserWorkloadsViewPath,
                      isActive: (location) => {
                          const search: string = location.search || '';
                          return search.includes(`filteredWorkflowView=Applications view`);
                      },
                      routeKey: 'violations',
                  },
                  {
                      type: 'link',
                      content: 'Platform',
                      path: violationsPlatformViewPath,
                      isActive: (location) => {
                          const search: string = location.search || '';
                          return search.includes(`filteredWorkflowView=Platform view`);
                      },
                      routeKey: 'violations',
                  },
                  {
                      type: 'link',
                      content: 'All Violations',
                      path: violationsFullViewPath,
                      isActive: (location) => {
                          const search: string = location.search || '';
                          return search.includes(`filteredWorkflowView=Full view`);
                      },
                      routeKey: 'violations',
                  },
              ]
            : [],
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
                              content: 'All vulnerable images',
                              description:
                                  'Findings for user, platform, and inactive images simultaneously',
                              path: vulnerabilitiesAllImagesPath,
                              routeKey: 'vulnerabilities/all-images',
                          },
                          {
                              type: 'link',
                              content: 'Inactive images',
                              description:
                                  'Findings for watched images and images not currently deployed as workloads based on your image retention settings',
                              path: vulnerabilitiesInactiveImagesPath,
                              routeKey: 'vulnerabilities/inactive-images',
                          },
                          {
                              type: 'link',
                              content: 'Images without CVEs',
                              description:
                                  'Images and workloads without observed CVEs (results might include false negatives due to scanner limitations, such as unsupported operating systems)',
                              path: vulnerabilitiesImagesWithoutCvesPath,
                              routeKey: 'vulnerabilities/images-without-cves',
                          },
                          {
                              type: 'link',
                              content: 'Kubernetes components',
                              description:
                                  'Vulnerabilities affecting the underlying Kubernetes infrastructure',
                              path: vulnerabilitiesPlatformCvesPath,
                              routeKey: 'vulnerabilities/platform-cves',
                          },
                      ],
                  },
              ]
            : [],
    };
}

// Since some subnav links may contain URL parameters, we need to strip these
// parameters off when determining whether or not to show a set of navigation items at the top.
// This is because react-router's `matchPath` does a strict comparison that wil always fail when
// our link's URL includes search parameters.
function matchBasePath({
    pathname,
    descriptionPath,
}: {
    pathname: string;
    descriptionPath: string;
}): boolean {
    const basePath = descriptionPath.split('?')[0] ?? '';
    return Boolean(matchPath({ path: `${basePath}/*` }, pathname));
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
                    return matchBasePath({ pathname, descriptionPath: group.path });
                }
                return group.children
                    .filter((child) => child.type === 'link')
                    .some(({ path }) => {
                        return matchBasePath({ pathname, descriptionPath: path });
                    });
            });
        }) ?? []
    );
}

export type HorizontalSubnavProps = {
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function HorizontalSubnav({ hasReadAccess, isFeatureFlagEnabled }: HorizontalSubnavProps) {
    const navigate = useNavigate();
    const location = useLocation();
    const routePredicates = { hasReadAccess, isFeatureFlagEnabled };

    const subnavDescriptionGroups = getSubnavDescriptionGroups(isFeatureFlagEnabled);
    const subnavDescriptionGroupForCurrentPath = getSubnavGroupForCurrentPath(
        subnavDescriptionGroups,
        location.pathname
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
        if (value !== undefined) {
            navigate(value.toString());
        }
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
                                    isActive={isActiveLink(location, subnavDescription)}
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
                            const activeChildLink = subnavDescription.children
                                .filter((c) => c.type === 'link')
                                .find((child) => isActiveLink(location, child));
                            const dropdownTitle = activeChildLink?.content ?? title;
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
                                            isActive={Boolean(activeChildLink)}
                                            onClick={() => onToggleClick(key)}
                                        >
                                            <MenuToggle
                                                ref={toggleRef}
                                                isExpanded={openDropdownKey === key}
                                                variant="plainText"
                                            >
                                                {typeof dropdownTitle === 'function'
                                                    ? dropdownTitle(subnavDescriptions)
                                                    : dropdownTitle}
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
                                            const isActive = isActiveLink(location, child);
                                            return (
                                                <DropdownItem
                                                    component={'a'}
                                                    className={
                                                        isActive
                                                            ? 'acs-pf-horizontal-subnav-menu__active'
                                                            : ''
                                                    }
                                                    isSelected={isActive}
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
