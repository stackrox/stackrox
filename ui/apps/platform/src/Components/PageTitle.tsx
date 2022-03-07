import React, { ReactElement } from 'react';
import { Helmet } from 'react-helmet';

import useBranding from 'hooks/useBranding';

type PageTitleProps = {
    title: string | null;
};

const PageTitle = ({ title }: PageTitleProps): ReactElement => {
    const branding = useBranding();
    const baseTitle = branding.basePageTitle;
    const text = title ? `${title} | ${baseTitle}` : baseTitle;
    return (
        <Helmet>
            <title>{text}</title>
        </Helmet>
    );
};

export default PageTitle;
