import React from 'react';
import PropTypes from 'prop-types';
import { HelpCircle } from 'react-feather';

import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';

const HelpIcon = ({ description }) => {
    return (
        <Tooltip content={<TooltipOverlay>{description}</TooltipOverlay>}>
            <HelpCircle className="h-4 w-4 text-tertiary-500" alt="help" />
        </Tooltip>
    );
};

HelpIcon.propTypes = {
    description: PropTypes.string.isRequired,
};

export default HelpIcon;
