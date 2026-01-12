import { useState } from 'react';
import type { Ref } from 'react';
import { useLocation, useNavigate } from 'react-router-dom-v5-compat';
import {
    Dropdown,
    DropdownItem,
    DropdownList,
    MenuToggle,
    NavItem,
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
import HorizontalSubnav from 'Components/Navigation/HorizontalSubnav';
import NavigationItem from 'Components/Navigation/NavigationItem';

const moreViewsItems = [
    {
        content: 'All vulnerable images',
        description: 'Findings for user, platform, and inactive images simultaneously',
        path: vulnerabilitiesAllImagesPath,
    },
    {
        content: 'Inactive images',
        description:
            'Findings for watched images and images not currently deployed as workloads based on your image retention settings',
        path: vulnerabilitiesInactiveImagesPath,
    },
    {
        content: 'Images without CVEs',
        description:
            'Images and workloads without observed CVEs (results might include false negatives due to scanner limitations, such as unsupported operating systems)',
        path: vulnerabilitiesImagesWithoutCvesPath,
    },
    {
        content: 'Kubernetes components',
        description: 'Vulnerabilities affecting the underlying Kubernetes infrastructure',
        path: vulnerabilitiesPlatformCvesPath,
    },
];

function VulnerabilitiesSubnav() {
    const navigate = useNavigate();
    const location = useLocation();
    const [isMoreViewsOpen, setIsMoreViewsOpen] = useState(false);

    const activeMoreViewItem = moreViewsItems.find((item) => location.pathname.includes(item.path));
    const dropdownTitle = activeMoreViewItem?.content ?? 'More Views';

    const onSelect = (_event: unknown, value: string | number | undefined) => {
        if (value !== undefined) {
            navigate(value.toString());
        }
        setIsMoreViewsOpen(false);
    };

    return (
        <HorizontalSubnav>
            <NavList>
                <NavigationItem
                    isActive={location.pathname.includes(vulnerabilitiesUserWorkloadsPath)}
                    path={vulnerabilitiesUserWorkloadsPath}
                    content="User Workloads"
                />
                <NavigationItem
                    isActive={location.pathname.includes(vulnerabilitiesPlatformPath)}
                    path={vulnerabilitiesPlatformPath}
                    content="Platform"
                />
                <NavigationItem
                    isActive={location.pathname.includes(vulnerabilitiesNodeCvesPath)}
                    path={vulnerabilitiesNodeCvesPath}
                    content="Nodes"
                />
                <NavigationItem
                    isActive={location.pathname.includes(vulnerabilitiesVirtualMachineCvesPath)}
                    path={vulnerabilitiesVirtualMachineCvesPath}
                    content="Virtual Machines"
                />
                <Dropdown
                    isPlain
                    onSelect={onSelect}
                    isOpen={isMoreViewsOpen}
                    onOpenChange={setIsMoreViewsOpen}
                    toggle={(toggleRef: Ref<MenuToggleElement>) => (
                        <NavItem
                            isActive={Boolean(activeMoreViewItem)}
                            onClick={() => setIsMoreViewsOpen(!isMoreViewsOpen)}
                        >
                            <MenuToggle
                                ref={toggleRef}
                                isExpanded={isMoreViewsOpen}
                                variant="plainText"
                            >
                                {dropdownTitle}
                            </MenuToggle>
                        </NavItem>
                    )}
                    shouldFocusToggleOnSelect
                >
                    <DropdownList>
                        {moreViewsItems.map((item) => {
                            const isActive = location.pathname.includes(item.path);
                            return (
                                <DropdownItem
                                    component="a"
                                    className={
                                        isActive ? 'acs-pf-horizontal-subnav-menu__active' : ''
                                    }
                                    isSelected={isActive}
                                    value={item.path}
                                    key={item.path}
                                    description={item.description}
                                >
                                    {item.content}
                                </DropdownItem>
                            );
                        })}
                    </DropdownList>
                </Dropdown>
            </NavList>
        </HorizontalSubnav>
    );
}

export default VulnerabilitiesSubnav;
