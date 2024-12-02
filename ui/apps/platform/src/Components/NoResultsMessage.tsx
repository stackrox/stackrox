import React, { ReactElement } from 'react';

export type NoResultsMessageProps = {
    message: string;
    className?: string;
};

function NoResultsMessage({ className, message }: NoResultsMessageProps): ReactElement {
    return (
        <div
            className={`flex flex-1 rounded items-center justify-center w-full leading-loose text-center h-full ${className}`}
        >
            {message}
        </div>
    );
}

export default NoResultsMessage;
