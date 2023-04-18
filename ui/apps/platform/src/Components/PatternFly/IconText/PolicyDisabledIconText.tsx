import React, { ReactNode } from 'react';
import { BanIcon, CheckIcon } from '@patternfly/react-icons';

import IconText from './IconText';

export type PolicyDisabledIconTextProps = {
    isDisabled: boolean;
    isTextOnly?: boolean;
};

function PolicyDisabledIconText({
    isDisabled,
    isTextOnly,
}: PolicyDisabledIconTextProps): ReactNode {
    const icon = isDisabled ? (
        <BanIcon />
    ) : (
        <CheckIcon color="var(--pf-global--success-color--100)" />
    );
    const text = isDisabled ? 'Disabled' : 'Enabled';

    return <IconText icon={icon} text={text} isTextOnly={isTextOnly} />;
}

export default PolicyDisabledIconText;
