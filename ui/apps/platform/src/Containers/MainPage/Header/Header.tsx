import React, { ReactElement } from 'react';
import {
    Masthead,
    MastheadBrand,
    MastheadContent,
    MastheadMain,
    MastheadToggle,
    PageToggleButton,
} from '@patternfly/react-core';
import { BarsIcon } from '@patternfly/react-icons';

import BrandLogo from 'Components/PatternFly/BrandLogo';
import MastheadToolbar from './MastheadToolbar';

function Header(): ReactElement {
    // PageToggleButton assumes isManagedSidebar prop of Page element.
    // aria-label="primary" prop makes header element a unique landmark.
    return (
        <Masthead className="ignore-react-onclickoutside theme-dark">
            <MastheadToggle>
                <PageToggleButton variant="plain">
                    <BarsIcon />
                </PageToggleButton>
            </MastheadToggle>
            <MastheadMain>
                <MastheadBrand>
                    <BrandLogo />
                </MastheadBrand>
            </MastheadMain>
            <MastheadContent className="pf-v5-u-flex-grow-1 pf-v5-u-justify-content-flex-end">
                <MastheadToolbar />
            </MastheadContent>
        </Masthead>
    );
}

export default Header;
