import type { ReactElement } from 'react';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { Icon } from '@patternfly/react-core';

import IconText from './IconText';

export type PolicyStatusIconTextProps = {
    isPass: boolean;
    isTextOnly?: boolean;
};

function PolicyStatusIconText({ isPass, isTextOnly }: PolicyStatusIconTextProps): ReactElement {
    const icon = isPass ? (
        <Icon>
            <CheckCircleIcon color="var(--pf-t--global--icon--color--status--success--default)" />
        </Icon>
    ) : (
        <Icon>
            <ExclamationCircleIcon color="var(--pf-t--global--icon--color--status--danger--default)" />
        </Icon>
    );
    const text = isPass ? 'Pass' : 'Fail';

    return <IconText icon={icon} text={text} isTextOnly={isTextOnly} />;
}

export default PolicyStatusIconText;
