import React, { ReactElement } from 'react';
import { Helmet } from 'react-helmet';
import { getProductBranding } from 'constants/productBranding';

const AppPageFavicon = (): ReactElement => {
    const branding = getProductBranding();
    return (
        <Helmet>
            <link rel="shortcut icon" href={branding.favicon} />
        </Helmet>
    );
};

export default AppPageFavicon;
