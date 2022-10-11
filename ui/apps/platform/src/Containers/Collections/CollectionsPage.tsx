import React, { useEffect } from 'react';
import { useParams } from 'react-router-dom';

import useURLParameter from 'hooks/useURLParameter';

import CollectionsTablePage from './CollectionsTablePage';
import CollectionsFormPage from './CollectionsFormPage';
import { parsePageActionProp } from './collections.utils';

function CollectionsPage() {
    // TODO Implement permissions once https://issues.redhat.com/browse/ROX-12619 is merged
    // const { hasWriteAccess } = usePermissions();
    const hasWriteAccessForCollections = true; // hasWriteAccess('TODO');

    const [pageAction, setPageAction] = useURLParameter('action', undefined);
    const { collectionId } = useParams();
    const validPageActionProp = parsePageActionProp(pageAction, collectionId);

    useEffect(() => {
        // If the URL structure somehow gets into a state with an invalid action, clear
        // the parameter to avoid confusing the user.
        if (typeof pageAction !== 'undefined' && !validPageActionProp) {
            setPageAction(undefined);
        }
    }, [pageAction, validPageActionProp, setPageAction]);

    if (validPageActionProp) {
        return (
            <CollectionsFormPage
                hasWriteAccessForCollections={hasWriteAccessForCollections}
                pageAction={validPageActionProp}
            />
        );
    }

    return <CollectionsTablePage hasWriteAccessForCollections={hasWriteAccessForCollections} />;
}

export default CollectionsPage;
