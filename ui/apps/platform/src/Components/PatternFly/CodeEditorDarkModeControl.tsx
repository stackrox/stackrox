import React from 'react';

import { CodeEditorControl } from '@patternfly/react-code-editor';
import { MoonIcon, SunIcon } from '@patternfly/react-icons';

export type CodeEditorDarkModeControlProps = {
    isDarkMode: boolean;
    onToggleDarkMode: () => void;
};

/**
 * A PatternFly code editor control that toggles dark mode.
 */
function CodeEditorDarkModeControl({
    isDarkMode,
    onToggleDarkMode,
}: CodeEditorDarkModeControlProps) {
    return (
        <CodeEditorControl
            icon={isDarkMode ? <SunIcon /> : <MoonIcon />}
            aria-label="Toggle code editor dark mode"
            tooltipProps={{
                content: isDarkMode ? 'Toggle to light mode' : 'Toggle to dark mode',
            }}
            onClick={onToggleDarkMode}
            isVisible
        />
    );
}

export default CodeEditorDarkModeControl;
