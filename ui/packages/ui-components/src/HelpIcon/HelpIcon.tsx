import React, { ReactElement } from 'react';
import { HelpCircle } from 'react-feather';

import Tooltip from '../Tooltip';
import TooltipOverlay from '../TooltipOverlay';

export type HelpIconProps = {
    description: string;
};

const HelpIcon = ({ description }: HelpIconProps): ReactElement => {
    return (
        <Tooltip content={<TooltipOverlay>{description}</TooltipOverlay>}>
            <HelpCircle
                className="h-4 w-4 text-primary-500"
                aria-label={description}
                data-testid="help-icon"
            />
        </Tooltip>
    );
};

export default HelpIcon;
