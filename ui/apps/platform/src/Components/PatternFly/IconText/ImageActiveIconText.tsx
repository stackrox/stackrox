import React, { ReactNode } from 'react';
import { CheckIcon, TimesIcon } from '@patternfly/react-icons';

import IconText from './IconText';

export type ImageActiveIconTextProps = {
    isActive: boolean;
    isTextOnly?: boolean;
};

function ImageActiveIconText({ isActive, isTextOnly }: ImageActiveIconTextProps): ReactNode {
    const Icon = isActive ? (
        <CheckIcon color="var(--pf-global--success-color--100)" />
    ) : (
        <TimesIcon />
    );
    const text = isActive ? 'Active' : 'Inactive';

    return <IconText Icon={Icon} text={text} isTextOnly={isTextOnly} />;
}

export default ImageActiveIconText;
