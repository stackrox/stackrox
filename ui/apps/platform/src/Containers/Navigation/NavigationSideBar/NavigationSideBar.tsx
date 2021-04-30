import React, { ReactElement } from 'react';
import { NavLink, useLocation, Location } from 'react-router-dom';
import { Nav, NavList, NavItem, NavExpandable, PageSidebar } from '@patternfly/react-core';

import { navItems } from './navigationUtils';

type LeftNavItemProps = {
    label: string;
    to: string | undefined;
    location: Location;
};

function LeftNavItem({ label, to, location }: LeftNavItemProps): ReactElement {
    return (
        <NavItem key={label} id={label} isActive={location.pathname.includes(to)}>
            <NavLink exact to={to} activeClassName="pf-m-current">
                {label}
            </NavLink>
        </NavItem>
    );
}

function NavigationSideBar(): ReactElement {
    const location: Location = useLocation();

    const Navigation = (
        <Nav id="nav-primary-simple">
            <NavList id="nav-list-simple">
                {navItems.map(({ isGrouped, children = [], label, to }) => {
                    if (isGrouped) {
                        return (
                            <NavExpandable
                                key={label}
                                id={label}
                                title={label}
                                isActive={children.some((navItem) =>
                                    location.pathname.includes(navItem.to)
                                )}
                            >
                                {children.map((navItem) => (
                                    <LeftNavItem
                                        label={navItem.label}
                                        to={navItem.to}
                                        location={location}
                                    />
                                ))}
                            </NavExpandable>
                        );
                    }
                    return <LeftNavItem label={label} to={to} location={location} />;
                })}
            </NavList>
        </Nav>
    );

    return <PageSidebar nav={Navigation} />;
}

export default NavigationSideBar;
