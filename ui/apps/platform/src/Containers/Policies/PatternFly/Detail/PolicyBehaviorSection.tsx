import React from 'react';
import { DescriptionList, Card, CardBody } from '@patternfly/react-core';

import { LifecycleStage, PolicyEventSource, EnforcementAction } from 'types/policy.proto';
import DescriptionListItem from 'Components/DescriptionListItem';
import {
    formatEventSource,
    formatLifecycleStages,
    formatResponse,
    getEnforcementLifecycleStages,
} from '../policies.utils';

type PolicyBehaviorSectionProps = {
    lifecycleStages: LifecycleStage[];
    eventSource: PolicyEventSource;
    enforcementActions: EnforcementAction[];
};

function PolicyBehaviorSection({
    lifecycleStages,
    eventSource,
    enforcementActions,
}: PolicyBehaviorSectionProps): React.ReactElement {
    const enforcementLifecycleStages = getEnforcementLifecycleStages(
        lifecycleStages,
        enforcementActions
    );
    return (
        <Card isFlat>
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
                </DescriptionList>
            </CardBody>
        </Card>
    );
}

export default PolicyBehaviorSection;
