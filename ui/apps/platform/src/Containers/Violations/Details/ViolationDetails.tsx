import type { ReactElement } from 'react';
import { Divider, Flex, FlexItem, Title } from '@patternfly/react-core';

import type { LifecycleStage } from 'types/policy.proto';
import type { FileAccessViolation, ProcessViolation, Violation } from 'types/alert.proto';

import DeploytimeMessages from './DeploytimeMessages';
import RuntimeMessages from './RuntimeMessages';

type ViolationDetailsProps = {
    processViolation: ProcessViolation | null;
    fileAccessViolation: FileAccessViolation | null;
    lifecycleStage: LifecycleStage;
    violations: Violation[];
};

function ViolationDetails({
    processViolation,
    fileAccessViolation,
    lifecycleStage,
    violations,
}: ViolationDetailsProps): ReactElement {
    const showRuntimeMessages =
        processViolation?.processes?.length ||
        fileAccessViolation?.accesses?.length ||
        lifecycleStage === 'RUNTIME';
    const showDeploytimeMessages = lifecycleStage === 'DEPLOY';
    return (
        <Flex>
            <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }}>
                <FlexItem>
                    <Title headingLevel="h2" className="pf-v6-u-mb-md">
                        Violation events
                    </Title>
                    <Divider component="div" />
                </FlexItem>
                {showRuntimeMessages && (
                    <FlexItem>
                        <RuntimeMessages
                            processViolation={processViolation}
                            fileAccessViolation={fileAccessViolation}
                            violations={violations}
                        />
                    </FlexItem>
                )}
                {showDeploytimeMessages && (
                    <FlexItem>
                        <DeploytimeMessages violations={violations} />
                    </FlexItem>
                )}
            </Flex>
        </Flex>
    );
}

export default ViolationDetails;
