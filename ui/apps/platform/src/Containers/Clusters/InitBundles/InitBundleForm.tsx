import React, { ReactElement } from 'react';

import { ClusterInitBundle } from 'services/ClustersService';

export type InitBundleFormProps = {
    initBundle: ClusterInitBundle | null;
};

function InitBundleForm({ initBundle }: InitBundleFormProps): ReactElement {
    return <>{initBundle?.name ?? ''}</>;
}

export default InitBundleForm;
