import { Stack, StackItem, Title } from '@patternfly/react-core';

import type { ProtoAdvisory } from './useCveDetail';
import AdvisoriesTable from './AdvisoriesTable';
import AffectedComponentsTable from './AffectedComponentsTable';
import AffectedImagesTable from './AffectedImagesTable';

type FlowLayoutProps = {
    advisories: ProtoAdvisory[];
};

/**
 * Renders all sections flowing vertically: advisories, components, images.
 */
function FlowLayout({ advisories }: FlowLayoutProps) {
    return (
        <Stack hasGutter>
            <StackItem>
                <Title headingLevel="h3">Advisories</Title>
                <AdvisoriesTable advisories={advisories} />
            </StackItem>
            <StackItem>
                <Title headingLevel="h3">Affected Components</Title>
                <AffectedComponentsTable />
            </StackItem>
            <StackItem>
                <Title headingLevel="h3">Affected Images</Title>
                <AffectedImagesTable />
            </StackItem>
        </Stack>
    );
}

export default FlowLayout;
