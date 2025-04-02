import React, { ReactElement } from 'react';
import {
    ClipboardCopyButton,
    CodeBlock,
    CodeBlockAction,
    CodeBlockCode,
} from '@patternfly/react-core';
import useClipboardCopy from 'hooks/useClipboardCopy';

export type ErrorBoundaryCodeBlockProps = {
    code: string;
    idForButton: string;
    idForContent: string;
    phraseForCopied: string;
    phraseForCopy: string;
};

function ErrorBoundaryCodeBlock({
    code,
    idForButton,
    idForContent,
    phraseForCopied,
    phraseForCopy,
}: ErrorBoundaryCodeBlockProps): ReactElement {
    const { wasCopied, copyToClipboard } = useClipboardCopy();

    const actions = (
        <CodeBlockAction>
            <ClipboardCopyButton
                aria-label={phraseForCopy}
                id={idForButton}
                onClick={() => copyToClipboard(code)}
                textId={idForContent}
                variant="plain"
            >
                {wasCopied ? phraseForCopied : phraseForCopy}
            </ClipboardCopyButton>
        </CodeBlockAction>
    );

    return (
        <CodeBlock actions={actions}>
            <CodeBlockCode>{code}</CodeBlockCode>
        </CodeBlock>
    );
}

export default ErrorBoundaryCodeBlock;
