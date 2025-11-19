import { Icon, Label, Tooltip } from '@patternfly/react-core';
import { FilterIcon } from '@patternfly/react-icons';
import type { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

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

export function DynamicTableLabel() {
    return (
        <Tooltip content="You are viewing a filtered set of table rows. Column values may also be changed to match the applied filters.">
            <Label color="blue" icon={<DynamicIcon />} isCompact>
                Filtered view
            </Label>
        </Tooltip>
    );
}
