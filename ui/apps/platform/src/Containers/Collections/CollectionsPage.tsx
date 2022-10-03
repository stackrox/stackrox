import React from 'react';

import CollectionsTablePage from './CollectionsTablePage';

function CollectionsPage() {
    // TODO Implement permissions once https://issues.redhat.com/browse/ROX-12619 is merged
    // const { hasWriteAccess } = usePermissions();
    const hasWriteAccessForCollections = true; // hasWriteAccess('TODO');

    return <CollectionsTablePage hasWriteAccessForCollections={hasWriteAccessForCollections} />;
}

export default CollectionsPage;
