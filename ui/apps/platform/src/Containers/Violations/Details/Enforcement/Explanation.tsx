import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Divider } from '@patternfly/react-core';

import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import { LifecycleStage } from 'types/policy.proto';

function getEnforcementExplanation(lifecycleStage: LifecycleStage, message: string) {
    if (lifecycleStage === LIFECYCLE_STAGES.DEPLOY) {
        return `Deployment data was evaluated against this security policy. ${message}`;
    }

    if (lifecycleStage === LIFECYCLE_STAGES.RUNTIME) {
        return `Runtime data was evaluated against this security policy. ${message}`;
    }

    return '';
}

type ExplanationProps = {
    policyId: string;
    lifecycleStage: LifecycleStage;
    enforcement: {
        message: string;
    };
};

function Explanation({ lifecycleStage, enforcement, policyId }: ExplanationProps): ReactElement {
    const linkAddr = `../policies/${policyId}`;

    return (
        <div className="pf-u-p-md" aria-label="Enforcement explanation message">
            <div className="pf-u-pb-md">
                {getEnforcementExplanation(lifecycleStage, enforcement.message)}
            </div>
            <Divider component="div" />
            <div className="pf-u-pt-md">
                If the enforcement action is being applied several times, learn more on how you can
                <Link to={linkAddr}> remediate and resolve the issue.</Link>
            </div>
        </div>
    );
}

export default Explanation;
