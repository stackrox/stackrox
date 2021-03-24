import React, { ReactElement } from 'react';
import { Check, X } from 'react-feather';

import { AccessType } from 'constants/accessControl';

const forbiddenIcon = <X className="text-alert-600 h-4 w-4" />;
const permittedIcon = <Check className="text-success-600 h-4 w-4" />;

export type AccessIconProps = {
    accessType: AccessType;
};

export function ReadAccessIcon({ accessType }: AccessIconProps): ReactElement {
    return accessType === 'NO_ACCESS' ? forbiddenIcon : permittedIcon;
}

export function WriteAccessIcon({ accessType }: AccessIconProps): ReactElement {
    return accessType === 'READ_WRITE_ACCESS' ? permittedIcon : forbiddenIcon;
}
