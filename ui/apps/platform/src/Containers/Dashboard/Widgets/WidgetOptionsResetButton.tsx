import React, { MouseEventHandler } from 'react';
import { Button, Tooltip } from '@patternfly/react-core';

import { UndoIcon } from '@patternfly/react-icons';

export type WidgetOptionsResetButtonProps = {
    onClick: MouseEventHandler<HTMLButtonElement>;
};

const buttonLabel = 'Revert to default options';

function WidgetOptionsResetButton({ onClick }: WidgetOptionsResetButtonProps) {
    return (
        <Tooltip content={buttonLabel}>
            <Button
                aria-label={buttonLabel}
                variant="plain"
                icon={<UndoIcon />}
                onClick={onClick}
            />
        </Tooltip>
    );
}

export default WidgetOptionsResetButton;
