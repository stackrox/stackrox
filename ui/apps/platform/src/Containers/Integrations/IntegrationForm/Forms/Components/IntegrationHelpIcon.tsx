import React, { ReactElement } from 'react';
import { Popover } from '@patternfly/react-core';
import { HelpIcon } from '@patternfly/react-icons';

import PopoverBodyContent from 'Components/PopoverBodyContent';

export type IntegrationHelpIconProps = {
    helpTitle: string;
    helpText: React.ReactElement;
    ariaLabel: string;
};

function IntegrationHelpIcon({
    helpTitle,
    helpText,
    ariaLabel,
}: IntegrationHelpIconProps): ReactElement {
    return (
        <Popover
            aria-label={helpTitle}
            bodyContent={<PopoverBodyContent headerContent={helpTitle} bodyContent={helpText} />}
        >
            <button
                type="button"
                aria-label={ariaLabel}
                onClick={(e) => e.preventDefault()}
                className="pf-v5-c-form__group-label-help"
            >
                <HelpIcon />
            </button>
        </Popover>
    );
}

export default IntegrationHelpIcon;
