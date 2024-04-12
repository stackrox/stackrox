import React, { ReactElement } from 'react';
import { ExclamationCircleIcon, ExclamationTriangleIcon } from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';
import { Icon } from '@patternfly/react-core';

// TODO import the following function components:

export const DangerIcon = (props: SVGIconProps) => (
    <Icon>
        <ExclamationCircleIcon color="var(--pf-v5-global--danger-color--100)" {...props} />
    </Icon>
);

export const WarningIcon = (props: SVGIconProps) => (
    <Icon>
        <ExclamationTriangleIcon color="var(--pf-v5-global--warning-color--100)" {...props} />
    </Icon>
);

type IconType = 'danger' | 'warning';

function getIcon(type?: IconType): ReactElement | null {
    const className = 'pf-v5-u-display-inline pf-v5-u-ml-sm';

    switch (type) {
        case 'danger':
            return <DangerIcon className={className} />;
        case 'warning':
            return <WarningIcon className={className} />;
        default:
            return null;
    }
}

export type TooltipFieldValueProps = {
    field: string;
    value: number | string;
    type?: IconType;
};

function TooltipFieldValue({ field, value, type }: TooltipFieldValueProps): ReactElement | null {
    if (value === null) {
        return null;
    }

    return (
        <div className="leading-normal">
            <span className="font-700">{field}: </span>
            <span>{value}</span>
            {getIcon(type)}
        </div>
    );
}

export default TooltipFieldValue;
