import React, { ReactElement } from 'react';
import { CodeBlock, CodeBlockCode, Flex } from '@patternfly/react-core';

import { AdministrationEvent } from 'services/AdministrationEventsService';

import AdministrationEventHint from './AdministrationEventHint';

export type AdministrationEventHintMessageProps = {
    event: AdministrationEvent;
};

function AdministrationEventHintMessage({
    event,
}: AdministrationEventHintMessageProps): ReactElement {
    const { hint, message } = event;

    return (
        <Flex direction={{ default: 'column' }}>
            {hint && <AdministrationEventHint hint={hint} />}
            <CodeBlock>
                <CodeBlockCode>{message}</CodeBlockCode>
            </CodeBlock>
        </Flex>
    );
}

export default AdministrationEventHintMessage;
