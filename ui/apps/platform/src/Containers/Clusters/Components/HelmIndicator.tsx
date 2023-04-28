import React, { ReactElement } from 'react';
import { Tooltip } from '@patternfly/react-core';

import { useTheme } from 'Containers/ThemeProvider';
import helm from 'images/helm.svg';

function HelmIndicator(): ReactElement {
    const { isDarkMode } = useTheme();
    const darkModeStyle = isDarkMode ? 'bg-base-800 rounded-full' : '';

    return (
        <Tooltip content="This cluster is managed by Helm.">
            <span className={`w-5 h-5 inline-block ${darkModeStyle}`}>
                <img className="w-5 h-5" src={helm} alt="Managed by Helm" />
            </span>
        </Tooltip>
    );
}

export default HelmIndicator;
