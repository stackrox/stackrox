import React, { ReactElement } from 'react';

type DeploytimeMessageProps = {
    message: string;
};

function DeploytimeMessage({ message }: DeploytimeMessageProps): ReactElement {
    return (
        <div className="mb-4 p-3 pb-2 shadow border border-base-200 text-base-600 flex justify-between leading-normal bg-base-100">
            {message}
        </div>
    );
}

type DeploytimeMessagesProps = {
    violations?: {
        message: string;
    }[];
};

function DeploytimeMessages({ violations = [] }: DeploytimeMessagesProps): ReactElement {
    return (
        <>
            {violations.map(({ message }) => (
                <DeploytimeMessage key={message} message={message} />
            ))}
        </>
    );
}

export default DeploytimeMessages;
