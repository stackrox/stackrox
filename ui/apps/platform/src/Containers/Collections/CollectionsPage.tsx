import React, { useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import usePermissions from 'hooks/usePermissions';
import useURLParameter from 'hooks/useURLParameter';

import CollectionsTablePage from './CollectionsTablePage';
import CollectionsFormPage from './CollectionsFormPage';
import { parsePageActionProp } from './collections.utils';

function CollectionsPage() {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForCollections = hasReadWriteAccess('WorkflowAdministration');

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
            <>
                <PageSection className="pf-u-h-100" padding={{ default: 'noPadding' }}>
                    <CollectionsFormPage
                        hasWriteAccessForCollections={hasWriteAccessForCollections}
                        pageAction={validPageActionProp}
                    />
                </PageSection>
            </>
        );
    }

    return <CollectionsTablePage hasWriteAccessForCollections={hasWriteAccessForCollections} />;
}

export default CollectionsPage;
