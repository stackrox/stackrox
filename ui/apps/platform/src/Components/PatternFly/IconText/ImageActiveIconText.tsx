import React, { ReactElement } from 'react';
import { CheckIcon, MinusIcon } from '@patternfly/react-icons';

import IconText from './IconText';

export type ImageActiveIconTextProps = {
    isActive: boolean;
    isTextOnly?: boolean;
};

function ImageActiveIconText({ isActive, isTextOnly }: ImageActiveIconTextProps): ReactElement {
    const icon = isActive ? <CheckIcon /> : <MinusIcon />;
    const text = isActive ? 'Active' : 'Inactive';

    return <IconText icon={icon} text={text} isTextOnly={isTextOnly} />;
}

export default ImageActiveIconText;
