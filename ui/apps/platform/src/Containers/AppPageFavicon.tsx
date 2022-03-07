import React, { ReactElement } from 'react';
import { Helmet } from 'react-helmet';

import useBranding from 'hooks/useBranding';

const AppPageFavicon = (): ReactElement => {
    const branding = useBranding();
    return (
        <Helmet>
            <link rel="shortcut icon" href={branding.favicon} />
        </Helmet>
    );
};

export default AppPageFavicon;
