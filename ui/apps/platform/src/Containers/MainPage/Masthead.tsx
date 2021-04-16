import React, { ReactElement } from 'react';
import { PageHeader } from '@patternfly/react-core';

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
            headerTools={<MastheadToolbar />}
            isNavOpen={isNavOpen}
            onNavToggle={onNavToggle}
        />
    );
}

export default Masthead;
