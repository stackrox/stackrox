import React, { ReactNode } from 'react';
import { Button, Popover, PopoverPosition } from '@patternfly/react-core';
import { CaretDownIcon } from '@patternfly/react-icons';

export type OptionsMenuProps = {
    bodyContent: ReactNode;
};

function WidgetOptionsMenu({ bodyContent }: OptionsMenuProps) {
    return (
        <Popover
            minWidth="0px"
            className="pf-u-pr-0"
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
