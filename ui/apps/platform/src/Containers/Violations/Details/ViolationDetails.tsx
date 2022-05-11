import React, { ReactElement } from 'react';
import { Divider, Flex, FlexItem, Title } from '@patternfly/react-core';

import ViolationTags from './AnalystNotes/ViolationTags';
import DeploytimeMessages from './DeploytimeMessages';
import RuntimeMessages from './RuntimeMessages';
import { ProcessViolation, LifecycleStage, Violation } from '../types/violationTypes';

type ViolationDetailsProps = {
    processViolation?: ProcessViolation;
    lifecycleStage: LifecycleStage;
    violationId: string;
    violations: Violation[];
};

function ViolationDetails({
    processViolation,
    lifecycleStage,
    violationId,
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
            <Flex
                direction={{ default: 'column' }}
                data-testid="violation-tags"
                flex={{ default: 'flex_1' }}
            >
                <FlexItem>
                    <Title headingLevel="h3" className="pf-u-mb-md">
                        Add violation metadata
                    </Title>
                    <Divider component="div" />
                </FlexItem>
                <FlexItem data-testid="violation-tags">
                    <ViolationTags resourceId={violationId} />
                </FlexItem>
            </Flex>
        </Flex>
    );
}

export default ViolationDetails;
