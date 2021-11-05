import React, { ReactElement } from 'react';

import { Policy } from 'types/policy.proto';

import { PoliciesAction } from '../policies.utils';

type PolicyWizardProps = {
    action: PoliciesAction;
    policy: Policy;
};

function PolicyWizard({ action, policy }: PolicyWizardProps): ReactElement {
    return (
        <>
            <div>{action}</div>
            <div>{policy.id}</div>
        </>
    );
}

export default PolicyWizard;
