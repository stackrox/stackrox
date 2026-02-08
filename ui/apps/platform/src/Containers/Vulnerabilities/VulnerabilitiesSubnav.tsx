import { useState } from 'react';
import type { MouseEvent as ReactMouseEvent, Ref } from 'react';
import { useLocation, useNavigate } from 'react-router-dom-v5-compat';
import {
    Dropdown,
    DropdownItem,
    DropdownList,
    MenuToggle,
    NavItem,
    NavItemSeparator,
    NavList,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';

import {
    vulnerabilitiesAllImagesPath,
    vulnerabilitiesImagesWithoutCvesPath,
    vulnerabilitiesInactiveImagesPath,
    vulnerabilitiesNodeCvesPath,
    vulnerabilitiesPlatformCvesPath,
    vulnerabilitiesPlatformPath,
    vulnerabilitiesUserWorkloadsPath,
    vulnerabilitiesVirtualMachineCvesPath,
} from 'routePaths';
import NavigationItem from 'Containers/MainPage/Navigation/NavigationItem';
import { filterNavDescriptions, isActiveLink } from 'Containers/MainPage/Navigation/utils';
import type { NavDescription } from 'Containers/MainPage/Navigation/utils';
import type { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import type { HasReadAccess } from 'hooks/usePermissions';
import { ensureExhaustive } from 'utils/type.utils';

type VulnerabilitiesSubnavProps = {
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function VulnerabilitiesSubnav({
    hasReadAccess,
    isFeatureFlagEnabled,
}: VulnerabilitiesSubnavProps) {
    const navigate = useNavigate();
    const location = useLocation();
    const [openDropdownKey, setOpenDropdownKey] = useState<string | null>(null);

    const navDescriptions: NavDescription[] = [
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
            type: 'link',
            content: 'Virtual Machines',
            path: vulnerabilitiesVirtualMachineCvesPath,
            routeKey: 'vulnerabilities/virtual-machine-cves',
        },
        {
            type: 'parent',
            key: 'More Views',
            title: 'More Views',
            children: [
                {
                    type: 'link',
                    content: 'All vulnerable images',
                    description: 'Findings for user, platform, and inactive images simultaneously',
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
    ];

    const filteredNavDescriptions = filterNavDescriptions(navDescriptions, {
        hasReadAccess,
        isFeatureFlagEnabled,
    });

    const onToggleClick = (key: string) => {
        setOpenDropdownKey((currentKey) => (currentKey === key ? null : key));
    };

    const onSelect = (
        _event: ReactMouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        if (value !== undefined) {
            navigate(value.toString());
        }
        setOpenDropdownKey(null);
    };

    return (
        <NavList>
            {filteredNavDescriptions.map((navDescription) => {
                switch (navDescription.type) {
                    case 'link': {
                        const { content, path } = navDescription;
                        return (
                            <NavigationItem
                                key={path}
                                isActive={isActiveLink(location, navDescription)}
                                path={path}
                                content={
                                    typeof content === 'function'
                                        ? content(filteredNavDescriptions)
                                        : content
                                }
                            />
                        );
                    }
                    case 'parent': {
                        const { key, title, children } = navDescription;
                        const activeChildLink = navDescription.children
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
                                toggle={(toggleRef: Ref<MenuToggleElement>) => (
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
                                                ? dropdownTitle(filteredNavDescriptions)
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
                                                <NavItemSeparator key={child.key} role="listitem" />
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
                                                    ? content(filteredNavDescriptions)
                                                    : content}
                                            </DropdownItem>
                                        );
                                    })}
                                </DropdownList>
                            </Dropdown>
                        );
                    }
                    default:
                        return ensureExhaustive(navDescription);
                }
            })}
        </NavList>
    );
}

export default VulnerabilitiesSubnav;
