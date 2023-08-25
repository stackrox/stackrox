import React, { ReactElement } from 'react';
import { Divider, Flex, FlexItem, Title } from '@patternfly/react-core';

import { LifecycleStage } from 'types/policy.proto';
import { ProcessViolation, Violation } from 'types/alert.proto';

import DeploytimeMessages from './DeploytimeMessages';
import RuntimeMessages from './RuntimeMessages';

type ViolationDetailsProps = {
    processViolation: ProcessViolation | null;
    lifecycleStage: LifecycleStage;
    violations: Violation[];
};

function ViolationDetails({
    processViolation,
    lifecycleStage,
    violations,
}: ViolationDetailsProps): ReactElement {
    const showRuntimeMessages = processViolation?.processes?.length || lifecycleStage === 'RUNTIME';
    const showDeploytimeMessages = lifecycleStage === 'DEPLOY';
    return (
        <Flex>
            <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }}>
                <FlexItem>
                    <Title headingLevel="h3" className="pf-u-mb-md">
                        Violation events
                    </Title>
                    <Divider component="div" />
                </FlexItem>
                {showRuntimeMessages && (
                    <FlexItem>
                        <RuntimeMessages
                            processViolation={processViolation}
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
