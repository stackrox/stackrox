import React from 'react';
import PropTypes from 'prop-types';
import { HelpCircle } from 'react-feather';

import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';

const HelpIcon = ({ description }) => {
    return (
        <Tooltip content={<TooltipOverlay>{description}</TooltipOverlay>}>
            <HelpCircle className="h-4 w-4 text-tertiary-500" />
        </Tooltip>
    );
};

HelpIcon.propTypes = {
    description: PropTypes.string.isRequired,
};

export default HelpIcon;
