import React, { ReactElement } from 'react';

import { ProcessViolation, Violation } from 'types/alert.proto';
import ProcessCard from './ProcessCard';
import NetworkFlowCard from './NetworkFlowCard';
import K8sCard from './K8sCard';

type RuntimeMessagesProps = {
    processViolation: ProcessViolation | null;
    violations?: Violation[];
};

function RuntimeMessages({ processViolation, violations }: RuntimeMessagesProps): ReactElement {
    const isPlainViolation = !!violations?.length;
    const plainViolations: ReactElement[] = [];

    violations?.forEach((violation) => {
        const { message } = violation;
        const time = violation.time ?? '';

        if (violation.type === 'NETWORK_FLOW') {
            const { networkFlowInfo } = violation;
            plainViolations.push(
                <NetworkFlowCard
                    key={`${time}-${message}`}
                    message={message}
                    networkFlowInfo={networkFlowInfo}
                    time={time}
                />
            );
        } else if (violation.type === 'K8S_EVENT') {
            const { keyValueAttrs } = violation;
            plainViolations.push(
                <K8sCard
                    key={`${time}-${message}`}
                    message={message}
                    keyValueAttrs={keyValueAttrs}
                    time={time}
                />
            );
        }
    });

    return (
        <>
            {isPlainViolation && plainViolations}
            {!!processViolation?.processes?.length && (
                <ProcessCard
                    processes={processViolation.processes}
                    message={processViolation.message}
                />
            )}
        </>
    );
}

export default RuntimeMessages;
