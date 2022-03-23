import React from 'react';
import { Divider, Flex, PageSection, Title } from '@patternfly/react-core';

import CIDRFormButton from './CIDRFormButton';
import FilterToolbar from './FilterToolbar';
import SimulatorButton from './SimulatorButton';

function Header({ isSimulationOn }) {
    return (
        <>
            <PageSection variant="light">
                <Flex direction={{ default: 'row' }}>
                    <Title className="pf-u-flex-grow-1" headingLevel="h1">
                        Network Graph
                    </Title>
                    <CIDRFormButton isDisabled={isSimulationOn} />
                    <SimulatorButton isDisabled={isSimulationOn} />
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <FilterToolbar isDisabled={isSimulationOn} />
            </PageSection>
            <Divider component="div" />
        </>
    );
}

export default Header;
