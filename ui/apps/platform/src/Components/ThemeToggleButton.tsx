import React, { ReactElement } from 'react';
import { Moon, Sun } from 'react-feather';
import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';

import { useTheme } from 'Containers/ThemeProvider';

const ThemeToggleButton = (): ReactElement => {
    const themeState = useTheme();
    const tooltipText = themeState.isDarkMode ? 'Switch to Light Mode' : 'Switch to Dark Mode';
    return (
        <Tooltip content={<TooltipOverlay>{tooltipText}</TooltipOverlay>}>
            <button
                aria-label="Invert theme"
                onClick={themeState.toggle}
                type="button"
                className="flex items-center pt-3 pb-2 px-4 no-underline rounded-l-sm"
            >
                <span>{themeState.isDarkMode ? <Sun size="16" /> : <Moon size="16" />}</span>
            </button>
        </Tooltip>
    );
};

export default ThemeToggleButton;
