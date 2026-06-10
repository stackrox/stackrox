import { Stack, StackItem, Title } from '@patternfly/react-core';

import type { ProtoAdvisory, ProtoComponent, ProtoImage } from './useCveDetail';
import AdvisoriesTable from './AdvisoriesTable';
import AffectedComponentsTable from './AffectedComponentsTable';
import AffectedImagesTable from './AffectedImagesTable';

type FlowLayoutProps = {
    advisories: ProtoAdvisory[];
    components: ProtoComponent[];
    images: ProtoImage[];
};

/**
 * Renders all sections flowing vertically: advisories, components, images.
 */
function FlowLayout({ advisories, components, images }: FlowLayoutProps) {
    return (
        <Stack hasGutter>
            <StackItem>
                <Title headingLevel="h3">Advisories</Title>
                <AdvisoriesTable advisories={advisories} />
            </StackItem>
            <StackItem>
                <Title headingLevel="h3">Affected Components</Title>
                <AffectedComponentsTable components={components} />
            </StackItem>
            <StackItem>
                <Title headingLevel="h3">Affected Images</Title>
                <AffectedImagesTable images={images} />
            </StackItem>
        </Stack>
    );
}

export default FlowLayout;
