import React from 'react';
import { Icon } from '@patternfly/react-core';
import { FilterIcon } from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

export function DynamicIcon(props: SVGIconProps) {
    return (
        <Icon>
            <FilterIcon color="var(--pf-v5-global--palette--blue-300)" {...props} />
        </Icon>
    );
}

export function DynamicColumnIcon() {
    return <DynamicIcon className="pf-v5-u-display-inline pf-v5-u-ml-sm" />;
}
