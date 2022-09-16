import React from 'react';

import NotFoundMessage from 'Components/NotFoundMessage';
import CollectionsTablePage from './CollectionsTablePage';

function CollectionsPage() {
    // TODO Implement permissions once https://issues.redhat.com/browse/ROX-12619 is merged
    // const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    const hasReadAccessForCollections = true; // hasReadAccess('TODO');
    const hasWriteAccessForCollections = true; // hasReadWriteAccess('TODO');

    if (!hasReadAccessForCollections) {
        return <NotFoundMessage title="404: We couldn't find that page" />;
    }

    return <CollectionsTablePage hasWriteAccessForCollections={hasWriteAccessForCollections} />;
}

export default CollectionsPage;
