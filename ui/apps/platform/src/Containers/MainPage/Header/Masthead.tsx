import React, { ReactElement } from 'react';
import { PageHeader } from '@patternfly/react-core';

import BrandLogo from 'Components/PatternFly/BrandLogo';
import MastheadToolbar from './MastheadToolbar';

function Masthead(): ReactElement {
    return (
        <PageHeader
            className="ignore-react-onclickoutside theme-dark"
            showNavToggle
            logo={<BrandLogo />}
            headerTools={<MastheadToolbar />}
        />
    );
}

export default Masthead;
