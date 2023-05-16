import React, { ReactElement } from 'react';
import { ExclamationCircleIcon, ExclamationTriangleIcon } from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

// TODO import the following function components:

export const DangerIcon = (props: SVGIconProps) => (
    <ExclamationCircleIcon color="var(--pf-global--danger-color--100)" {...props} />
);

export const WarningIcon = (props: SVGIconProps) => (
    <ExclamationTriangleIcon color="var(--pf-global--warning-color--100)" {...props} />
);

type IconType = 'danger' | 'warning';

function getIcon(type?: IconType): ReactElement | null {
    const className = 'pf-u-display-inline pf-u-ml-sm';

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
