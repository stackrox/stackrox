import React, { ReactElement } from 'react';

import { Policy } from 'types/policy.proto';

type PolicyDetailProps = {
    policy: Policy;
};

function PolicyDetail({ policy }: PolicyDetailProps): ReactElement {
    return (
        <>
            <div>view</div>
            <div>{policy.id}</div>
        </>
    );
}

export default PolicyDetail;
