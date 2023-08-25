import React, { ReactElement } from 'react';
import { MinusIcon } from '@patternfly/react-icons';

import IconText from './IconText';

export type NotApplicableIconTextProps = {
    isTextOnly?: boolean;
};

function NotApplicableIconText({ isTextOnly }: NotApplicableIconTextProps): ReactElement {
    const icon = <MinusIcon />;
    const text = 'N/A';

    return <IconText icon={icon} text={text} isTextOnly={isTextOnly} />;
}

export default NotApplicableIconText;
