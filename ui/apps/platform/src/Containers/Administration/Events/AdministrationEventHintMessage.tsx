import React, { ReactElement } from 'react';
import { CodeBlock, Flex } from '@patternfly/react-core';

import { AdministrationEvent } from 'services/AdministrationEventsService';

export type AdministrationEventHintMessageProps = {
    event: AdministrationEvent;
};

function AdministrationEventHintMessage({
    event,
}: AdministrationEventHintMessageProps): ReactElement {
    const { hint, message } = event;

    return (
        <Flex direction={{ default: 'column' }}>
            {hint && (
                <div>
                    {hint.split('\n').map((line) => (
                        <p key={line}>{line}</p>
                    ))}
                </div>
            )}
            <CodeBlock>{message}</CodeBlock>
        </Flex>
    );
}

export default AdministrationEventHintMessage;
