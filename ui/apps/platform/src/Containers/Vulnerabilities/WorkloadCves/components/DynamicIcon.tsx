import React from 'react';
import { Label, Tooltip } from '@patternfly/react-core';
import { BoltIcon } from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

export function DynamicIcon(props: SVGIconProps) {
    return <BoltIcon color="var(--pf-global--palette--blue-300)" {...props} />;
}

export function DynamicColumnIcon() {
    return <DynamicIcon className="pf-u-display-inline pf-u-ml-sm" />;
}

export function DynamicTableLabel() {
    return (
        <Tooltip content="You are viewing a filtered set of table rows. Column values may also be changed to match the applied filters.">
            <Label color="blue" icon={<DynamicIcon />}>
                Filtered view
            </Label>
        </Tooltip>
    );
}
