import type { ReactElement } from 'react';
import { Button, Tooltip } from '@patternfly/react-core';
import { MoonIcon, SunIcon } from '@patternfly/react-icons';

import { useTheme } from 'hooks/useTheme';

function ThemeToggleButton(): ReactElement {
    const { isDarkMode, toggle } = useTheme();
    const tooltipText = isDarkMode ? 'Switch to Light Mode' : 'Switch to Dark Mode';

    return (
        <Tooltip content={<div>{tooltipText}</div>} position="bottom">
            <Button aria-label="Toggle theme" onClick={toggle} variant="plain">
                {isDarkMode ? <SunIcon /> : <MoonIcon />}
            </Button>
        </Tooltip>
    );
}

export default ThemeToggleButton;
