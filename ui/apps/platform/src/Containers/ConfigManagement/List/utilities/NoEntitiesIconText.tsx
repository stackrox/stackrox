import type { ReactElement } from 'react';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';
import { Icon } from '@patternfly/react-core';

import IconText from 'Components/PatternFly/IconText/IconText';

export type NoEntitiesIconTextProps = {
    text: string;
};

/*
 * Render No Whatevers text with icon only when it is a security problem.
 * Otherwise, render as plain text.
 */
function NoEntitiesIconText({ text }: NoEntitiesIconTextProps): ReactElement {
    const icon = (
        <Icon>
            <ExclamationTriangleIcon color="var(--pf-t--global--icon--color--status--warning--default)" />
        </Icon>
    );

    return <IconText icon={icon} text={text} />;
}

export default NoEntitiesIconText;
