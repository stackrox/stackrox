import React, { ReactElement } from 'react';
import uniqBy from 'lodash/uniqBy';
import { Flex, FlexItem, Divider, Card } from '@patternfly/react-core';

import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import { ENFORCEMENT_ACTIONS } from 'constants/enforcementActions';
import { Alert } from 'types/alert.proto';
import Header from './Enforcement/Header';
import Explanation from './Enforcement/Explanation';

function getRuntimeEnforcementCount(processViolation) {
    return uniqBy(processViolation.processes, 'podId').length;
}

type EnforcementDetailsProps = {
    alert: Alert;
    enforcement: NonNullable<Alert['enforcement']>;
};

function EnforcementDetails({ alert, enforcement }: EnforcementDetailsProps): ReactElement {
    const { lifecycleStage, processViolation, policy } = alert;
    let enforcementCount = 0;
    if (lifecycleStage === LIFECYCLE_STAGES.RUNTIME) {
        if (enforcement.action === ENFORCEMENT_ACTIONS.KILL_POD_ENFORCEMENT) {
            enforcementCount = processViolation?.processes
                ? getRuntimeEnforcementCount(processViolation)
                : 0;
        } else {
            enforcementCount = 1;
        }
    } else if (lifecycleStage === LIFECYCLE_STAGES.DEPLOY) {
        enforcementCount = 1;
    }

    return (
        <Card>
            <Flex direction={{ default: 'column' }}>
                <FlexItem>
                    <Header
                        lifecycleStage={alert.lifecycleStage}
                        enforcementCount={enforcementCount}
                        enforcementAction={enforcement.action}
                    />
                    {enforcementCount && (
                        <>
                            <Divider component="div" inset={{ default: 'insetMd' }} />
                            <Explanation
                                lifecycleStage={lifecycleStage}
                                enforcement={enforcement}
                                policyId={policy.id}
                            />
                        </>
                    )}
                </FlexItem>
            </Flex>
        </Card>
    );
}

export default EnforcementDetails;
