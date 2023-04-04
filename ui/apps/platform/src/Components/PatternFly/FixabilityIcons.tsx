import React from 'react';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

export const FixableIcon = (props: SVGIconProps) => (
    <CheckCircleIcon color="var(--pf-global--success-color--100)" {...props} />
);

export const NotFixableIcon = (props: SVGIconProps) => (
    <ExclamationCircleIcon color="var(--pf-global--danger-color--100)" {...props} />
);
