import React, { ReactElement } from 'react';
import { CheckIcon, MinusIcon } from '@patternfly/react-icons';

import IconText from './IconText';

export type PolicyDisabledIconTextProps = {
    isDisabled: boolean;
    isTextOnly?: boolean;
};

function PolicyDisabledIconText({
    isDisabled,
    isTextOnly,
}: PolicyDisabledIconTextProps): ReactElement {
    const icon = isDisabled ? <MinusIcon /> : <CheckIcon />;
    const text = isDisabled ? 'Disabled' : 'Enabled';

    return <IconText icon={icon} text={text} isTextOnly={isTextOnly} />;
}

export default PolicyDisabledIconText;
