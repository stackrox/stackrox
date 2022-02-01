import React, { ReactElement } from 'react';

import { ProcessViolation, Violation } from '../types/violationTypes';
import ProcessCard from './ProcessCard';
import NetworkFlowCard from './NetworkFlowCard';
import K8sCard from './K8sCard';

type RuntimeMessagesProps = {
    processViolation?: ProcessViolation;
    violations?: Violation[];
};

function RuntimeMessages({ processViolation, violations }: RuntimeMessagesProps): ReactElement {
    const isPlainViolation = !!violations?.length;
    const plainViolations: ReactElement[] = [];

    violations?.forEach(({ message, networkFlowInfo, time, keyValueAttrs }) => {
        if (networkFlowInfo) {
            plainViolations.push(
                <NetworkFlowCard
                    key={`${time}-${message}`}
                    message={message}
                    networkFlowInfo={networkFlowInfo}
                    time={time}
                />
            );
        } else {
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
