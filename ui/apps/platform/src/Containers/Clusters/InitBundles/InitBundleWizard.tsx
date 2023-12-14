import React, { ReactElement } from 'react';

import InitBundlesHeader from './InitBundlesHeader';

function InitBundleWizard(): ReactElement {
    return (
        <>
            <InitBundlesHeader titleNotInitBundles="Create bundle" />;
        </>
    );
}

export default InitBundleWizard;
