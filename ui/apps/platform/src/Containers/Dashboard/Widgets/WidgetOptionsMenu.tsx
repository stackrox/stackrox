import React, { ReactNode } from 'react';
import { Button, Popover, PopoverPosition } from '@patternfly/react-core';
import { CaretDownIcon, CogIcon } from '@patternfly/react-icons';

import './WidgetOptionsMenu.css';

export type OptionsMenuProps = {
    bodyContent: ReactNode;
};

function WidgetOptionsMenu({ bodyContent }: OptionsMenuProps) {
    return (
        <Popover
            aria-label="Options menu"
            className="widget-options-menu"
            minWidth="0px"
            position={PopoverPosition.bottomEnd}
            enableFlip={false}
            bodyContent={bodyContent}
        >
            <Button
                aria-label="Options"
                variant="secondary"
                className="pf-v5-u-mr-sm"
                icon={<CaretDownIcon />}
                iconPosition="right"
            >
                <CogIcon className="pf-v5-u-display-inline" />
            </Button>
        </Popover>
    );
}

export default WidgetOptionsMenu;
