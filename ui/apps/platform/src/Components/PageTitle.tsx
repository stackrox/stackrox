import React, { ReactElement } from 'react';
import { Helmet } from 'react-helmet';

type PageTitleProps = {
    title: string | null;
};

const baseTitle = 'Red Hat Advanced Cluster Security';

const PageTitle = ({ title }: PageTitleProps): ReactElement => {
    const text = title ? `${title} | ${baseTitle}` : baseTitle;
    return (
        <Helmet>
            <title>{text}</title>
        </Helmet>
    );
};

export default PageTitle;
