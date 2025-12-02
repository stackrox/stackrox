import type { ReactElement } from 'react';
import {
    Masthead,
    MastheadBrand,
    MastheadContent,
    MastheadLogo,
    MastheadMain,
    MastheadToggle,
    PageToggleButton,
} from '@patternfly/react-core';

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
        <Masthead className="ignore-react-onclickoutside" inset={{ default: 'insetNone' }}>
            <Notifications />
            <PublicConfigHeader />
            <Banners />

            <MastheadMain>
                <MastheadToggle className="pf-v6-u-pl-lg">
                    <PageToggleButton isHamburgerButton variant="plain"></PageToggleButton>
                </MastheadToggle>
                <MastheadBrand data-codemods>
                    <MastheadLogo data-codemods>
                        <BrandLogo />
                    </MastheadLogo>
                </MastheadBrand>
            </MastheadMain>
            <MastheadContent className="pf-v6-u-flex-grow-1 pf-v6-u-justify-content-flex-end pf-v6-u-pr-lg">
                <MastheadToolbar />
            </MastheadContent>
        </Masthead>
    );
}

export default Header;
