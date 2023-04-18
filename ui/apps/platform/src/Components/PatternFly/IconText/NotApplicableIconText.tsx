import React, { ReactElement } from 'react';
import { Minus } from 'react-feather';

import IconText from './IconText';

export type NotApplicableIconTextProps = {
    isTextOnly?: boolean;
};

function NotApplicableIconText({ isTextOnly }: NotApplicableIconTextProps): ReactElement {
    const icon = <Minus className="h-4 w-4" />;
    const text = 'N/A';

    return <IconText icon={icon} text={text} isTextOnly={isTextOnly} />;
}

export default NotApplicableIconText;
