import React, { ReactElement } from 'react';

import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';
import { useTheme } from 'Containers/ThemeProvider';
import operatorLogo from 'images/operator-logo.png';

function OperatorIndicator(): ReactElement {
    const { isDarkMode } = useTheme();
    const darkModeStyle = isDarkMode ? 'bg-base-800 rounded-full' : '';

    return (
        <Tooltip
            content={
                <TooltipOverlay>This cluster is managed by a Kubernetes Operator.</TooltipOverlay>
            }
        >
            <span className={`w-5 h-5 inline-block ${darkModeStyle}`}>
                <img
                    className="w-5 h-5"
                    src={operatorLogo}
                    alt="Managed by a Kubernetes Operator"
                />
            </span>
        </Tooltip>
    );
}

export default OperatorIndicator;
