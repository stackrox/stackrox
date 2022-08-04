import React, { ReactElement } from 'react';
import { NavLink } from 'react-router-dom';
import { NavItem } from '@patternfly/react-core';

export type LeftNavItemProps = {
    isActive: boolean;
    path: string;
    title: string;
};

function LeftNavItem({ isActive, path, title }: LeftNavItemProps): ReactElement {
    return (
        <NavItem id={title} isActive={isActive}>
            <NavLink exact to={path} activeClassName="pf-m-current">
                {title}
            </NavLink>
        </NavItem>
    );
}

export default LeftNavItem;
