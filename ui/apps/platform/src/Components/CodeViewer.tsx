import React, { CSSProperties, ReactNode, useState } from 'react';
import {
    CodeBlockAction,
    ClipboardCopyButton,
    Button,
    CodeBlock,
    CodeBlockCode,
} from '@patternfly/react-core';
import { MoonIcon, SunIcon } from '@patternfly/react-icons';

import useClipboardCopy from 'hooks/useClipboardCopy';

export type CodeViewerProps = {
    code: string;
    className?: string;
    style?: CSSProperties;
    additionalControls?: ReactNode;
};

const defaultStyle = {
    '--pf-v5-u-max-height--MaxHeight': '300px',
    overflowY: 'scroll',
} as const;

export default function CodeViewer({
    code,
    className = 'pf-v5-u-max-height',
    style,
    additionalControls,
}: CodeViewerProps) {
    const { wasCopied, setWasCopied, copyToClipboard } = useClipboardCopy();

    const [isDarkMode, setIsDarkMode] = useState(false);

    function toggleDarkMode() {
        setIsDarkMode((prevValue) => !prevValue);
    }

    const actions = (
        <>
            <CodeBlockAction>
                <ClipboardCopyButton
                    id="copy-code-button"
                    textId="copy-code-button"
                    aria-label="Copy code to clipboard"
                    onClick={() => copyToClipboard(code)}
                    exitDelay={wasCopied ? 1500 : 600}
                    variant="plain"
                    onTooltipHidden={() => setWasCopied(false)}
                >
                    {wasCopied ? 'Successfully copied to clipboard!' : 'Copy to clipboard'}
                </ClipboardCopyButton>
            </CodeBlockAction>
            <CodeBlockAction>
                <Button
                    variant="plain"
                    aria-label={isDarkMode ? 'Set light mode' : 'Set dark mode'}
                    icon={isDarkMode ? <SunIcon /> : <MoonIcon />}
                    onClick={() => toggleDarkMode()}
                />
            </CodeBlockAction>
            {additionalControls}
        </>
    );

    return (
        <CodeBlock
            className={`${isDarkMode ? 'pf-v5-theme-dark' : ''} ${className}`}
            style={{ ...defaultStyle, ...style }}
            actions={actions}
        >
            {/* TODO - Once Tailwind is gone we will need to remove this font size override */}
            <CodeBlockCode className="pf-v5-u-font-size-xs">{code}</CodeBlockCode>
        </CodeBlock>
    );
}
