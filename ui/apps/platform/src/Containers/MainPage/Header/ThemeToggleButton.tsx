import type { CSSProperties, ReactElement } from 'react';
import { Button, Tooltip } from '@patternfly/react-core';
import { MoonIcon, SunIcon } from '@patternfly/react-icons';

import { useTheme } from 'hooks/useTheme';

// On masthead, black text on white background like a dropdown menu.
const styleTooltip = {
    '--pf-v6-c-tooltip__content--Color': 'var(--pf-t--global--text--color--regular)',
    '--pf-v6-c-tooltip__content--BackgroundColor':
        'var(--pf-t--global--background--color--primary--default)',
} as CSSProperties;

function ThemeToggleButton(): ReactElement {
    const { isDarkMode, toggle } = useTheme();
    const tooltipText = isDarkMode ? 'Switch to Light Mode' : 'Switch to Dark Mode';

    return (
        <Tooltip content={<div>{tooltipText}</div>} position="bottom" style={styleTooltip}>
            <Button aria-label="Toggle theme" onClick={toggle} variant="plain">
                {isDarkMode ? <SunIcon /> : <MoonIcon />}
            </Button>
        </Tooltip>
    );
}

export default ThemeToggleButton;
