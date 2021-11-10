import React, { ReactElement } from 'react';

import { Policy } from 'types/policy.proto';

import { PageAction } from '../policies.utils';

type PolicyWizardProps = {
    pageAction: PageAction;
    policy: Policy;
};

function PolicyWizard({ pageAction, policy }: PolicyWizardProps): ReactElement {
    return (
        <>
            <div>{pageAction}</div>
            <div>{policy.id}</div>
        </>
    );
}

export default PolicyWizard;
