import React, { ReactElement } from 'react';
import { Brand, PageHeader } from '@patternfly/react-core';

import rhacsLogo from 'images/RHACS-Logo.svg';
import MastheadToolbar from './MastheadToolbar';

export type MastheadProps = {
    isNavOpen: boolean;
    onNavToggle: () => void;
};

function Masthead({ isNavOpen, onNavToggle }: MastheadProps): ReactElement {
    return (
        <PageHeader
            className="z-20 ignore-react-onclickoutside theme-dark"
            showNavToggle
            logo={<Brand src={rhacsLogo} alt="Red Hat Advanced Cluster Security" />}
            headerTools={<MastheadToolbar />}
            isNavOpen={isNavOpen}
            onNavToggle={onNavToggle}
        />
    );
}

export default Masthead;
