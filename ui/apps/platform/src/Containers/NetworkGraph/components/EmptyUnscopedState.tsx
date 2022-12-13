import React from 'react';
import { Bullseye } from '@patternfly/react-core';
import { ModuleIcon } from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';

function EmptyIcon(props: SVGIconProps) {
    return (
        <ModuleIcon {...props} size="lg" style={{ color: 'var(--pf-global--palette--red-100)' }} />
    );
}

function EmptyUnscopedState() {
    return (
        <Bullseye>
            <EmptyStateTemplate
                title="Select a cluster and at least one namespace to render active deployment traffic
                    on the graph"
                headingLevel="h2"
                icon={EmptyIcon}
            />
        </Bullseye>
    );
}

export default EmptyUnscopedState;
