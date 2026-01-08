import type { ReactElement } from 'react';
import { Nav } from '@patternfly/react-core';

import { useSubnavContent } from './SubnavContext';

import './HorizontalSubnav.css';

function HorizontalSubnav(): ReactElement | null {
    const { content } = useSubnavContent();

    if (!content) {
        return null;
    }

    return (
        <Nav variant="horizontal-subnav" className="acs-pf-horizontal-subnav">
            {content}
        </Nav>
    );
}

export default HorizontalSubnav;
