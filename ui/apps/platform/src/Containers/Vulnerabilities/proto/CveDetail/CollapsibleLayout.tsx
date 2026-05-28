import { useState } from 'react';
import { ExpandableSection, Stack, StackItem } from '@patternfly/react-core';

import type { ProtoAdvisory } from './useCveDetail';
import AdvisoriesTable from './AdvisoriesTable';
import AffectedComponentsTable from './AffectedComponentsTable';
import AffectedImagesTable from './AffectedImagesTable';

type CollapsibleLayoutProps = {
    advisories: ProtoAdvisory[];
};

/**
 * All sections rendered as PatternFly ExpandableSection.
 * Advisories expanded by default; Components and Images collapsed.
 */
function CollapsibleLayout({ advisories }: CollapsibleLayoutProps) {
    const [advisoriesExpanded, setAdvisoriesExpanded] = useState(true);
    const [componentsExpanded, setComponentsExpanded] = useState(false);
    const [imagesExpanded, setImagesExpanded] = useState(false);

    return (
        <Stack hasGutter>
            <StackItem>
                <ExpandableSection
                    toggleText={`Advisories (${advisories.length})`}
                    isExpanded={advisoriesExpanded}
                    onToggle={(_event, expanded) => setAdvisoriesExpanded(expanded)}
                >
                    <AdvisoriesTable advisories={advisories} />
                </ExpandableSection>
            </StackItem>
            <StackItem>
                <ExpandableSection
                    toggleText="Affected Components"
                    isExpanded={componentsExpanded}
                    onToggle={(_event, expanded) => setComponentsExpanded(expanded)}
                >
                    <AffectedComponentsTable />
                </ExpandableSection>
            </StackItem>
            <StackItem>
                <ExpandableSection
                    toggleText="Affected Images"
                    isExpanded={imagesExpanded}
                    onToggle={(_event, expanded) => setImagesExpanded(expanded)}
                >
                    <AffectedImagesTable />
                </ExpandableSection>
            </StackItem>
        </Stack>
    );
}

export default CollapsibleLayout;
