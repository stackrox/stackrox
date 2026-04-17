import type { ReactElement } from 'react';
import { Divider, Flex, PageSection, Title } from '@patternfly/react-core';

// import type { DetailsType } from '../reports.types';

function DetailsStep(): ReactElement {
    return (
        <PageSection>
            <Flex direction={{ default: 'column' }}>
                <Title headingLevel="h2">Details</Title>
                <Divider component="div" />
                {/* TODO Form includes name and description */}
            </Flex>
        </PageSection>
    );
}

export default DetailsStep;
