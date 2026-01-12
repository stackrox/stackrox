import type { ReactNode } from 'react';
import { Nav } from '@patternfly/react-core';

import './HorizontalSubnav.css';

type HorizontalSubnavProps = {
    children: ReactNode;
};

function HorizontalSubnav({ children }: HorizontalSubnavProps) {
    return (
        <Nav variant="horizontal-subnav" className="acs-pf-horizontal-subnav">
            {children}
        </Nav>
    );
}

export default HorizontalSubnav;
