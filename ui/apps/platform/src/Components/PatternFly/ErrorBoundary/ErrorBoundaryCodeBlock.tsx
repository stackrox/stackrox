import React, { ReactElement, useState } from 'react';
import {
    ClipboardCopyButton,
    CodeBlock,
    CodeBlockAction,
    CodeBlockCode,
} from '@patternfly/react-core';

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
    const [wasCopied, setWasCopied] = useState(false);

    function onClickCopy() {
        // https://developer.mozilla.org/en-US/docs/Web/API/Clipboard/writeText#browser_compatibility
        // Chrome 66 Edge 79 Firefox 63 Safari 13.1
        navigator?.clipboard?.writeText(code).then(() => {
            setWasCopied(true);
        });
    }

    const actions = (
        <CodeBlockAction>
            <ClipboardCopyButton
                aria-label={phraseForCopy}
                id={idForButton}
                onClick={onClickCopy}
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
