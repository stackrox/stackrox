import React from 'react';
import usePublicConfig from 'hooks/usePublicConfig';

export default function LoginNotice() {
    const { publicConfig } = usePublicConfig();
    const loginNotice = publicConfig?.loginNotice;

    if (!loginNotice || !loginNotice.text || !loginNotice.enabled) {
        return null;
    }

    return (
        <div
            className="flex w-full justify-center border-t h-43 overflow-auto"
            data-testid="login-notice"
        >
            <div className="whitespace-pre-wrap leading-normal">
                <div className="px-8 py-5">{loginNotice.text}</div>
            </div>
        </div>
    );
}
