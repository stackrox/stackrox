import React, {
    createContext,
    CSSProperties,
    Dispatch,
    ReactNode,
    SetStateAction,
    useContext,
    useState,
} from 'react';
import { CodeBlockAction, ClipboardCopyButton, Button, CodeBlock } from '@patternfly/react-core';
import { MoonIcon, SunIcon } from '@patternfly/react-icons';

import { PrismLight as SyntaxHighlighter } from 'react-syntax-highlighter';
import lightTheme from 'react-syntax-highlighter/dist/esm/styles/prism/one-light';
import darkTheme from 'react-syntax-highlighter/dist/esm/styles/prism/one-dark';
import yaml from 'react-syntax-highlighter/dist/esm/languages/prism/yaml';

import useClipboardCopy from 'hooks/useClipboardCopy';

SyntaxHighlighter.registerLanguage('yaml', yaml);

const CodeViewerThemeContext = createContext<
    ['light' | 'dark', Dispatch<SetStateAction<'light' | 'dark'>>] | undefined
>(undefined);

export const CodeViewerThemeProvider = ({ children }) => {
    const [state, setState] = useState<'light' | 'dark'>('light');

    return (
        <CodeViewerThemeContext.Provider value={[state, setState]}>
            {children}
        </CodeViewerThemeContext.Provider>
    );
};

export const useCodeViewerThemeContext = () => {
    const context = useContext(CodeViewerThemeContext);
    // Fallback state provides the ability to toggle theme for a single instance if no provider is detected uptree
    const fallbackState = useState<'light' | 'dark'>('light');
    return context ?? fallbackState;
};

// When adding to the supported languages, the correct language definition must be imported and registered as well
type SupportedLanguages = 'yaml';

const defaultStyle = {
    '--pf-v5-u-max-height--MaxHeight': '300px',
    '--pf-v5-c-code-block__content--PaddingTop': '0',
    '--pf-v5-c-code-block__content--PaddingBottom': '0',
    '--pf-v5-c-code-block__content--PaddingLeft': '0',
    '--pf-v5-c-code-block__content--PaddingRight': '0',
    overflowY: 'auto',
} as const;

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
}: CodeViewerProps) {
    const { wasCopied, setWasCopied, copyToClipboard } = useClipboardCopy();
    const [theme, setTheme] = useCodeViewerThemeContext();

<<<<<<< HEAD
    function toggleTheme() {
=======
    function toggleDarkMode() {
>>>>>>> 7101b07f8f (Add syntax highlighting support)
        setTheme((prevValue) => (prevValue === 'light' ? 'dark' : 'light'));
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
                    aria-label={theme === 'light' ? 'Set dark theme' : 'Set light theme'}
                    icon={theme === 'light' ? <MoonIcon /> : <SunIcon />}
                    onClick={() => toggleTheme()}
                />
            </CodeBlockAction>
            {additionalControls}
        </>
    );

    // TODO - When Tailwind is removed, we likely need to get rid of this font size override
    return (
        <CodeBlock
            className={`${theme === 'light' ? '' : 'pf-v5-theme-dark'} pf-v5-u-p-0 pf-v5-u-font-size-xs pf-v5-u-max-height ${className}`}
            style={{ ...defaultStyle, ...style }}
            actions={actions}
        >
            <SyntaxHighlighter
                language={language}
                showLineNumbers
                wrapLongLines
                style={theme === 'light' ? lightTheme : darkTheme}
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
