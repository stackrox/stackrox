import type { ReactElement } from 'react';

import type { FileAccessViolation, ProcessViolation, Violation } from 'types/alert.proto';
import NetworkFlowCard from './NetworkFlowCard';
import K8sCard from './K8sCard';
import TimestampedEventCard from './TimestampedEventCard';
import ProcessCardContent from './ProcessCardContent';
import FileAccessCardContent from './FileAccessCardContent';

type RuntimeMessagesProps = {
    processViolation: ProcessViolation | null;
    fileAccessViolation: FileAccessViolation | null;
    violations?: Violation[];
};

function RuntimeMessages({
    processViolation,
    fileAccessViolation,
    violations,
}: RuntimeMessagesProps): ReactElement {
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
        } else if (violation.type === 'FILE_ACCESS') {
            const { fileAccess } = violation;
            plainViolations.push(
                <TimestampedEventCard
                    message={message}
                    events={fileAccess}
                    getTimestamp={(access) => access.timestamp}
                    ContentComponent={FileAccessCardContent}
                    getEventKey={(access) =>
                        `${access.timestamp}-${access.operation}-${access.file.actualPath}`
                    }
                />
            );
        }
    });

    return (
        <>
            {isPlainViolation && plainViolations}
            {!!processViolation?.processes?.length && (
                <TimestampedEventCard
                    message={processViolation.message}
                    events={processViolation.processes}
                    getTimestamp={(process) => process.signal.time}
                    getEventKey={(process) => process.signal.id}
                    ContentComponent={ProcessCardContent}
                />
            )}
        </>
    );
}

export default RuntimeMessages;
