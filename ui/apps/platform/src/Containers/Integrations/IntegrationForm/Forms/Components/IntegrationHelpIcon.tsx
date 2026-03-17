import type { ReactElement } from 'react';
import { FormGroupLabelHelp, Popover } from '@patternfly/react-core';

import PopoverBodyContent from 'Components/PopoverBodyContent';

export type IntegrationHelpIconProps = {
    helpTitle: string;
    helpText: ReactElement;
    ariaLabel: string;
    hasAutoWidth?: boolean;
};

function IntegrationHelpIcon({
    helpTitle,
    helpText,
    ariaLabel,
    hasAutoWidth,
}: IntegrationHelpIconProps): ReactElement {
    return (
        <Popover
            aria-label={helpTitle}
            bodyContent={<PopoverBodyContent headerContent={helpTitle} bodyContent={helpText} />}
            hasAutoWidth={hasAutoWidth}
        >
            <FormGroupLabelHelp aria-label={ariaLabel} />
        </Popover>
    );
}

export default IntegrationHelpIcon;
