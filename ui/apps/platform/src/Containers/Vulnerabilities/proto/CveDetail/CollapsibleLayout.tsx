import { useState } from 'react';
import { ExpandableSection, Stack, StackItem } from '@patternfly/react-core';

import type { ProtoAdvisory, ProtoComponent, ProtoImage } from './useCveDetail';
import AdvisoriesTable from './AdvisoriesTable';
import AffectedComponentsTable from './AffectedComponentsTable';
import AffectedImagesTable from './AffectedImagesTable';

type CollapsibleLayoutProps = {
    advisories: ProtoAdvisory[];
    components: ProtoComponent[];
    images: ProtoImage[];
};

/**
 * All sections rendered as PatternFly ExpandableSection.
 * Advisories expanded by default; Components and Images collapsed.
 */
function CollapsibleLayout({
    advisories,
    components,
    images,
}: CollapsibleLayoutProps) {
    const [advisoriesExpanded, setAdvisoriesExpanded] = useState(true);
    const [componentsExpanded, setComponentsExpanded] = useState(false);
    const [imagesExpanded, setImagesExpanded] = useState(false);

    return (
        <Stack hasGutter>
            <StackItem>
                <ExpandableSection
                    toggleText={`Advisories (${advisories.length})`}
                    isExpanded={advisoriesExpanded}
                    onToggle={(_event, expanded) =>
                        setAdvisoriesExpanded(expanded)
                    }
                >
                    <AdvisoriesTable advisories={advisories} />
                </ExpandableSection>
            </StackItem>
            <StackItem>
                <ExpandableSection
                    toggleText={`Affected Components (${components.length})`}
                    isExpanded={componentsExpanded}
                    onToggle={(_event, expanded) =>
                        setComponentsExpanded(expanded)
                    }
                >
                    <AffectedComponentsTable components={components} />
                </ExpandableSection>
            </StackItem>
            <StackItem>
                <ExpandableSection
                    toggleText={`Affected Images (${images.length})`}
                    isExpanded={imagesExpanded}
                    onToggle={(_event, expanded) =>
                        setImagesExpanded(expanded)
                    }
                >
                    <AffectedImagesTable images={images} />
                </ExpandableSection>
            </StackItem>
        </Stack>
    );
}

export default CollapsibleLayout;
