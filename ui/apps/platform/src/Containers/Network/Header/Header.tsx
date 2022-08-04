import React, { ReactElement } from 'react';
import { Divider, Flex, PageSection, Title } from '@patternfly/react-core';

import CIDRFormButton from './CIDRFormButton';
import FilterToolbar from './FilterToolbar';
import SimulatorButton from './SimulatorButton';

type HeaderProps = {
    isGraphDisabled: boolean;
    isSimulationOn: boolean;
};

function Header({ isGraphDisabled, isSimulationOn }: HeaderProps): ReactElement {
    return (
        <>
            <PageSection variant="light">
                <Flex direction={{ default: 'row' }}>
                    <Title className="pf-u-flex-grow-1" headingLevel="h1">
                        Network Graph
                    </Title>
                    <CIDRFormButton isDisabled={isGraphDisabled || isSimulationOn} />
                    <SimulatorButton isDisabled={isGraphDisabled || isSimulationOn} />
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
