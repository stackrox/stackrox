import React, { ReactElement } from 'react';
import { useLocation, useParams } from 'react-router-dom';
import qs from 'qs';

import usePermissions from 'hooks/usePermissions';

import InitBundlePage from './InitBundlePage';
import InitBundlesPage from './InitBundlesPage';
import InitBundleWizard from './InitBundleWizard';

function hasCreateAction(search: string) {
    const { action } = qs.parse(search, { ignoreQueryPrefix: true });
    return action === 'create';
}

function InitBundlesRoute(): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    // TODO replace resources with Admin role.
    const hasWriteAccessForInitBundles =
        hasReadWriteAccess('Administration') && hasReadWriteAccess('Integration');

    const { search } = useLocation();
    const { id } = useParams(); // see clustersInitBundlesPathWithParam in routePaths.ts

    const isCreateAction = hasWriteAccessForInitBundles && hasCreateAction(search);

    if (id) {
        return (
            <InitBundlePage hasWriteAccessForInitBundles={hasWriteAccessForInitBundles} id={id} />
        );
    }

    if (hasWriteAccessForInitBundles && isCreateAction) {
        return <InitBundleWizard />;
    }

    return <InitBundlesPage hasWriteAccessForInitBundles={hasWriteAccessForInitBundles} />;
}

export default InitBundlesRoute;
