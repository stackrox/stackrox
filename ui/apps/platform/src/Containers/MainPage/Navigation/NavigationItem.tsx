import React, { ReactElement } from 'react';
import { NavLink } from 'react-router-dom-v5-compat';
import { NavItem } from '@patternfly/react-core';

export type NavigationItemProps = {
    isActive: boolean;
    path: string;
    content: string | ReactElement;
};

function NavigationItem({ isActive, path, content }: NavigationItemProps): ReactElement {
    return (
        <NavItem isActive={isActive}>
            <NavLink to={path} className={({ isActive }) => (isActive ? 'pf-m-current' : '')} end>
                {content}
            </NavLink>
        </NavItem>
    );
}

export default NavigationItem;
