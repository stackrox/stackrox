import React, { ReactElement } from 'react';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { Icon } from '@patternfly/react-core';

import IconText from './IconText';

export type PolicyStatusIconTextProps = {
    isPass: boolean;
    isTextOnly?: boolean;
};

function PolicyStatusIconText({ isPass, isTextOnly }: PolicyStatusIconTextProps): ReactElement {
    const icon = isPass ? (
        <Icon color="var(--pf-global--success-color--100)">
            <CheckCircleIcon />
        </Icon>
    ) : (
        <Icon color="var(--pf-global--danger-color--100)">
            <ExclamationCircleIcon />
        </Icon>
    );
    const text = isPass ? 'Pass' : 'Fail';

    return <IconText icon={icon} text={text} isTextOnly={isTextOnly} />;
}

export default PolicyStatusIconText;
