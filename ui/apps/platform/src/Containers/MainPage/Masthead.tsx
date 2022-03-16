import React, { ReactElement } from 'react';
import { Brand, PageHeader } from '@patternfly/react-core';

import rhacsLogo from 'images/RHACS-Logo.svg';
import MastheadToolbar from './MastheadToolbar';

function Masthead(): ReactElement {
    return (
        <PageHeader
            className="ignore-react-onclickoutside theme-dark"
            showNavToggle
            logo={<Brand src={rhacsLogo} alt="Red Hat Advanced Cluster Security" />}
            headerTools={<MastheadToolbar />}
        />
    );
}

export default Masthead;
