import React, { ReactElement } from 'react';
import { ThumbsDown, ThumbsUp } from 'react-feather';

import IconText from './IconText';

export type PolicyStatusIconTextProps = {
    isPass: boolean;
    isTextOnly?: boolean;
};

function PolicyStatusIconText({ isPass, isTextOnly }: PolicyStatusIconTextProps): ReactElement {
    const icon = isPass ? (
        <ThumbsUp className="h-4 w-4 pf-u-success-color-100" />
    ) : (
        <ThumbsDown className="h-4 w-4 pf-u-danger-color-100" />
    );
    const text = isPass ? 'Pass' : 'Fail';

    return <IconText icon={icon} text={text} isTextOnly={isTextOnly} />;
}

export default PolicyStatusIconText;
