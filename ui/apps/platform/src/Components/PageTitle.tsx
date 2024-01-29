import React, { ReactElement } from 'react';
import { Helmet } from 'react-helmet';

import { getProductBranding } from 'constants/productBranding';

type PageTitleProps = {
    title: string | null | undefined;
};

const PageTitle = ({ title }: PageTitleProps): ReactElement => {
    const branding = getProductBranding();
    const baseTitle = branding.basePageTitle;
    const text = title ? `${title} | ${baseTitle}` : baseTitle;
    return (
        <Helmet>
            <title>{text}</title>
        </Helmet>
    );
};

export default PageTitle;
