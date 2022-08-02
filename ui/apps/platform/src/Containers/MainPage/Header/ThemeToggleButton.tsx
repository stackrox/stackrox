import React, { CSSProperties, ReactElement } from 'react';
import { Moon, Sun } from 'react-feather';
import { Button, Tooltip } from '@patternfly/react-core';

import { useTheme } from 'Containers/ThemeProvider';

// On masthead, black text on white background like a dropdown menu.
const styleTooltip = {
    '--pf-c-tooltip__content--Color': 'var(--pf-global--Color--100)',
    '--pf-c-tooltip__content--BackgroundColor': 'var(--pf-global--BackgroundColor--100)',
} as CSSProperties;

const ThemeToggleButton = (): ReactElement => {
    const themeState = useTheme();
    const tooltipText = themeState.isDarkMode ? 'Switch to Light Mode' : 'Switch to Dark Mode';

    return (
        <Tooltip content={<div>{tooltipText}</div>} position="bottom" style={styleTooltip}>
            <Button aria-label="Invert theme" onClick={themeState.toggle} variant="plain">
                {themeState.isDarkMode ? <Sun size="16" /> : <Moon size="16" />}
            </Button>
        </Tooltip>
    );
};

export default ThemeToggleButton;
