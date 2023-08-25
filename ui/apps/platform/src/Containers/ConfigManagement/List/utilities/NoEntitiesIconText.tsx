import React, { ReactElement } from 'react';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';

import IconText from 'Components/PatternFly/IconText/IconText';

export type NoEntitiesIconTextProps = {
    text: string;
    isTextOnly?: boolean;
};

/*
 * Render No Whatevers text with icon only when it is a security problem.
 * Otherwise, render as plain text.
 */
function NoEntitiesIconText({ text, isTextOnly }: NoEntitiesIconTextProps): ReactElement {
    const icon = <ExclamationTriangleIcon color="var(--pf-global--warning-color--100)" />;

    return <IconText icon={icon} text={text} isTextOnly={isTextOnly} />;
}

export default NoEntitiesIconText;
