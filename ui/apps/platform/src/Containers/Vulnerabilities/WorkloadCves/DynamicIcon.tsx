import React from 'react';
import { Label, Tooltip } from '@patternfly/react-core';
import { BoltIcon } from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

export function DynamicIcon(props: SVGIconProps) {
    return <BoltIcon color="var(--pf-global--palette--blue-300)" {...props} />;
}

export function DynamicColumnIcon() {
    return (
        <Tooltip content="Data in this column can change according to the applied filters">
            <DynamicIcon className="pf-u-display-inline pf-u-ml-sm" />
        </Tooltip>
    );
}

export function DynamicTableLabel() {
    return (
        <Label isCompact color="blue" icon={<DynamicIcon />}>
            Filtered view
        </Label>
    );
}
