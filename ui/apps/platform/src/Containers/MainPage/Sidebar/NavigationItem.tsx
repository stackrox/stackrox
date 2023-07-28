import React, { ReactElement } from 'react';
import { NavLink } from 'react-router-dom';
import { NavItem } from '@patternfly/react-core';

export type NavigationItemProps = {
    isActive: boolean;
    path: string;
    content: string | ReactElement;
};

function NavigationItem({ isActive, path, content }: NavigationItemProps): ReactElement {
    return (
        <NavItem isActive={isActive}>
            <NavLink exact to={path} activeClassName="pf-m-current">
                {content}
            </NavLink>
        </NavItem>
    );
}

export default NavigationItem;
