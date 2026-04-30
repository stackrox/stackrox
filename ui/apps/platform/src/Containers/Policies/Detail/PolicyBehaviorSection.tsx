import type { ReactElement } from 'react';
import { Card, CardBody, DescriptionList } from '@patternfly/react-core';

import type {
    EnforcementAction,
    EvaluationFilter,
    LifecycleStage,
    PolicyEventSource,
} from 'types/policy.proto';
import DescriptionListItem from 'Components/DescriptionListItem';
import {
    formatEventSource,
    formatLifecycleStages,
    formatResponse,
    getEnforcementLifecycleStages,
} from '../policies.utils';

function formatImageLayerFilter(filter?: EvaluationFilter): string {
    if (!filter || filter.skipImageLayers === 'SKIP_NONE') {
        return 'All layers';
    }
    if (filter.skipImageLayers === 'SKIP_APP') {
        return 'Base image layers only';
    }
    if (filter.skipImageLayers === 'SKIP_BASE') {
        return 'Application layers only';
    }
    return 'All layers';
}

type PolicyBehaviorSectionProps = {
    lifecycleStages: LifecycleStage[];
    eventSource: PolicyEventSource;
    enforcementActions: EnforcementAction[];
    evaluationFilter?: EvaluationFilter;
};

function PolicyBehaviorSection({
    lifecycleStages,
    eventSource,
    enforcementActions,
    evaluationFilter,
}: PolicyBehaviorSectionProps): ReactElement {
    const enforcementLifecycleStages = getEnforcementLifecycleStages(
        lifecycleStages,
        enforcementActions
    );
    return (
        <Card>
            <CardBody>
                <DescriptionList isCompact isHorizontal>
                    <DescriptionListItem
                        term="Lifecycle stages"
                        desc={formatLifecycleStages(lifecycleStages)}
                    />
                    <DescriptionListItem
                        term="Event source"
                        desc={formatEventSource(eventSource)}
                    />
                    <DescriptionListItem
                        term="Response"
                        desc={formatResponse(enforcementLifecycleStages)}
                    />
                    {enforcementLifecycleStages?.length !== 0 && (
                        <DescriptionListItem
                            term="Enforcement"
                            desc={formatLifecycleStages(enforcementLifecycleStages)}
                        />
                    )}
                    <DescriptionListItem
                        term="Image layer filter"
                        desc={formatImageLayerFilter(evaluationFilter)}
                    />
                </DescriptionList>
            </CardBody>
        </Card>
    );
}

export default PolicyBehaviorSection;
