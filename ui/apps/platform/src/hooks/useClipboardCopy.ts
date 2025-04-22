import { Dispatch, SetStateAction, useState } from 'react';

export type UseClipboardCopyReturn = {
    wasCopied: boolean;
    setWasCopied: Dispatch<SetStateAction<boolean>>;
    error: unknown;
    copyToClipboard: (text: string) => Promise<void>;
};

/**
 * Hook that provides shorthand for copying text to the browser's clipboard
 */
export default function useClipboardCopy(): UseClipboardCopyReturn {
    const [wasCopied, setWasCopied] = useState(false);
    const [error, setError] = useState<unknown>();

    function copyToClipboard(text: string) {
        return navigator.clipboard
            .writeText(text)
            .then(() => {
                setWasCopied(true);
            })
            .catch(setError);
    }
    return { wasCopied, setWasCopied, error, copyToClipboard };
}
