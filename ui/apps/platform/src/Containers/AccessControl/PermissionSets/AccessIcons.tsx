import React, { ReactElement } from 'react';
import { CheckIcon, TimesIcon } from '@patternfly/react-icons';

import { AccessLevel } from 'services/RolesService';

const forbiddenIcon = (
    <TimesIcon aria-label="forbidden" color="var(--pf-global--danger-color--100)" size="sm" />
);
const permittedIcon = (
    <CheckIcon aria-label="permitted" color="var(--pf-global--success-color--100)" size="sm" />
);

export type AccessIconProps = {
    accessType: AccessLevel;
};

export function ReadAccessIcon({ accessType }: AccessIconProps): ReactElement {
    return accessType === 'NO_ACCESS' ? forbiddenIcon : permittedIcon;
}

export function WriteAccessIcon({ accessType }: AccessIconProps): ReactElement {
    return accessType === 'READ_WRITE_ACCESS' ? permittedIcon : forbiddenIcon;
}
