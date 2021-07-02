import React, { ReactElement } from 'react';
import { NavLink, Location } from 'react-router-dom';
import { NavItem } from '@patternfly/react-core';

import { basePathToLabelMap } from 'routePaths';

export type LeftNavItemProps = {
    location: Location;
    path: string;
};

function LeftNavItem({ location, path }: LeftNavItemProps): ReactElement {
    const label = basePathToLabelMap[path];
    return (
        <NavItem id={label} isActive={location.pathname.includes(path)}>
            <NavLink exact to={path} activeClassName="pf-m-current">
                {label}
            </NavLink>
        </NavItem>
    );
}

export default LeftNavItem;
