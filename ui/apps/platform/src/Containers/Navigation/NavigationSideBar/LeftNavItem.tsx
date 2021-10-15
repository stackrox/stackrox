import React, { ReactElement } from 'react';
import { NavLink } from 'react-router-dom';
import { NavItem } from '@patternfly/react-core';

import { basePathToLabelMap } from 'routePaths';

export type LeftNavItemProps = {
    isActive: boolean;
    path: string;
};

function LeftNavItem({ isActive, path }: LeftNavItemProps): ReactElement {
    const label = basePathToLabelMap[path];
    return (
        <NavItem id={label} isActive={isActive}>
            <NavLink exact to={path} activeClassName="pf-m-current">
                {label}
            </NavLink>
        </NavItem>
    );
}

export default LeftNavItem;
