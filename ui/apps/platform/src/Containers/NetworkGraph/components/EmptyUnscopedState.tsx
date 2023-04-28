import React from 'react';
import { Bullseye, Text } from '@patternfly/react-core';
import { ModuleIcon } from '@patternfly/react-icons';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';

function EmptyIcon(props: SVGIconProps) {
    return <ModuleIcon {...props} size="lg" />;
}

function EmptyUnscopedState() {
    return (
        <Bullseye>
            <EmptyStateTemplate title="Nothing to render yet" headingLevel="h2" icon={EmptyIcon}>
                <Text>
                    Select a cluster and at least one namespace to render
                    <br /> active deployment traffic on the graph
                </Text>
            </EmptyStateTemplate>
        </Bullseye>
    );
}

export default EmptyUnscopedState;
