import type { CSSProperties, ReactElement, ReactNode } from 'react';
import { ClipboardCopyButton, CodeBlock, CodeBlockAction } from '@patternfly/react-core';

import { PrismLight as SyntaxHighlighter } from 'react-syntax-highlighter';
import lightTheme from 'react-syntax-highlighter/dist/esm/styles/prism/one-light';
import darkTheme from 'react-syntax-highlighter/dist/esm/styles/prism/one-dark';
import yaml from 'react-syntax-highlighter/dist/esm/languages/prism/yaml';

import useClipboardCopy from 'hooks/useClipboardCopy';
import { useTheme } from 'hooks/useTheme';

SyntaxHighlighter.registerLanguage('yaml', yaml);

// When adding to the supported languages, the correct language definition must be imported and registered as well
type SupportedLanguages = 'yaml';

export type CodeViewerProps = {
    code: string;
    language?: SupportedLanguages;
    className?: string;
    style?: CSSProperties;
    additionalControls?: ReactNode;
};

export default function CodeViewer({
    code,
    language = 'yaml',
    className = '',
    style,
    additionalControls,
}: CodeViewerProps): ReactElement {
    const { wasCopied, setWasCopied, copyToClipboard } = useClipboardCopy();
    const theme = useTheme();

    const actions = (
        <>
            <CodeBlockAction>
                <ClipboardCopyButton
                    id="copy-code-button"
                    aria-label="Copy code to clipboard"
                    onClick={() => copyToClipboard(code)}
                    exitDelay={wasCopied ? 1500 : 600}
                    variant="plain"
                    onTooltipHidden={() => setWasCopied(false)}
                >
                    {wasCopied ? 'Successfully copied to clipboard!' : 'Copy to clipboard'}
                </ClipboardCopyButton>
            </CodeBlockAction>
            {additionalControls}
        </>
    );

    // TODO - When Tailwind is removed, we likely need to get rid of this font size override
    return (
        <CodeBlock
            className={`pf-v6-u-p-0 pf-v6-u-font-size-xs pf-v6-u-max-height ${className}`}
            style={style}
            actions={actions}
        >
            <SyntaxHighlighter
                language={language}
                showLineNumbers
                wrapLongLines
                style={theme.isDarkMode ? darkTheme : lightTheme}
                customStyle={{
                    margin: 0,
                    background: 'var(--pf-v6-c-code-block--BackgroundColor)',
                }}
            >
                {code}
            </SyntaxHighlighter>
        </CodeBlock>
    );
}
