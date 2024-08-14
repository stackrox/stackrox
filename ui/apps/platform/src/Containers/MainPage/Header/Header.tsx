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
import Banners from '../Banners/Banners';
import PublicConfigHeader from '../PublicConfig/PublicConfigHeader';
import MastheadToolbar from './MastheadToolbar';
import Notifications from './Notifications';

// Style rule for Notifications, PublicConfigHeader, and Banners elements.
import './Header.css';

function Header(): ReactElement {
    // PageToggleButton assumes isManagedSidebar prop of Page element.
    // aria-label="primary" prop makes header element a unique landmark.
    return (
        <Masthead
            className="ignore-react-onclickoutside theme-dark"
            inset={{ default: 'insetNone' }}
        >
            <Notifications />
            <PublicConfigHeader />
            <Banners />
            <MastheadToggle className="pf-v5-u-pl-lg">
                <PageToggleButton variant="plain">
                    <BarsIcon />
                </PageToggleButton>
            </MastheadToggle>
            <MastheadMain>
                <MastheadBrand>
                    <BrandLogo />
                </MastheadBrand>
            </MastheadMain>
            <MastheadContent className="pf-v5-u-flex-grow-1 pf-v5-u-justify-content-flex-end pf-v5-u-pr-lg">
                <MastheadToolbar />
            </MastheadContent>
        </Masthead>
    );
}

export default Header;
