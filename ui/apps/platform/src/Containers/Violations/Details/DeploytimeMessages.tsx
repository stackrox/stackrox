import React from 'react';
import type { ReactElement } from 'react';
import { Card, CardBody } from '@patternfly/react-core';

type DeploytimeMessageProps = {
    message: string;
};

function DeploytimeMessage({ message }: DeploytimeMessageProps): ReactElement {
    return (
        <Card isFlat className="pf-v5-u-mb-md">
            <CardBody>{message}</CardBody>
        </Card>
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
