import React, { ReactElement, ReactNode } from 'react';

import { Message } from '@stackrox/ui-components';

export type MessageCenteredProps = {
    children: ReactNode;
    type: 'base' | 'success' | 'warn' | 'error';
};

/*
 * Display children in a message box
 * that is centered in the full height and width of its parent.
 */
const MessageCentered = ({ children, type }: MessageCenteredProps): ReactElement => (
    <div className="flex h-full items-center justify-center w-full">
        <div className="m-6 w-full md:w-1/2 xl:w-3/5">
            <Message type={type}>{children}</Message>
        </div>
    </div>
);

export default MessageCentered;
