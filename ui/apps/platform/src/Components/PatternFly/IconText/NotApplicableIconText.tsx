import React, { ReactElement } from 'react';
import { Minus } from 'react-feather';

import IconText from './IconText';

export type NotApplicableIconTextProps = {
    isTextOnly?: boolean;
};

function NotApplicableIconText({ isTextOnly }: NotApplicableIconTextProps): ReactElement {
    const Icon = <Minus className="h-4 w-4" />;
    const text = 'N/A';

    return <IconText Icon={Icon} text={text} isTextOnly={isTextOnly} />;
}

export default NotApplicableIconText;
