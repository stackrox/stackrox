import React, { ReactElement } from 'react';

import { ToggleButtonProps } from '../ToggleButton';

type ToggleButtonChild = {
    props: ToggleButtonProps;
};

export type ToggleButtonGroupProps = {
    activeToggleButton: string;
    children: ToggleButtonChild[];
};

// @TODO: See if we can replace the usage of the RadioButtonGroup Component for boolean policy logic with this one
function ToggleButtonGroup({ activeToggleButton, children }: ToggleButtonGroupProps): ReactElement {
    const modifiedChildren = React.Children.map(children, (child: ToggleButtonChild) => {
        if (React.isValidElement(child)) {
            return React.cloneElement(child, {
                isActive: activeToggleButton === child.props.value,
            });
        }
        return child;
    });
    return <div>{modifiedChildren}</div>;
}

export default ToggleButtonGroup;
