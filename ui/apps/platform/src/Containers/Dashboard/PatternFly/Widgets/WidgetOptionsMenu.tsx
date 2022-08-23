import React, { ReactNode } from 'react';
import { Button, Popover, PopoverPosition } from '@patternfly/react-core';
import { CaretDownIcon } from '@patternfly/react-icons';

import './WidgetOptionsMenu.css';

export type OptionsMenuProps = {
    bodyContent: ReactNode;
};

function WidgetOptionsMenu({ bodyContent }: OptionsMenuProps) {
    return (
        <Popover
            className="widget-options-menu"
            minWidth="0px"
            position={PopoverPosition.bottomEnd}
            enableFlip={false}
            bodyContent={bodyContent}
        >
            <Button
                variant="secondary"
                className="pf-u-mr-sm"
                icon={<CaretDownIcon />}
                iconPosition="right"
            >
                Options
            </Button>
        </Popover>
    );
}

export default WidgetOptionsMenu;
