import * as React from 'react';
import { PageSection, Title } from '@patternfly/react-core';
import { CheckCircleIcon } from '@patternfly/react-icons';

import SeverityCountLabels from 'Containers/Vulnerabilities/components/SeverityCountLabels';

export function Index() {
    return (
        <>
            <PageSection>
                <Title headingLevel="h1">{'Hello, Plugin!'}</Title>
                <SeverityCountLabels
                    criticalCount={10}
                    importantCount={20}
                    moderateCount={30}
                    lowCount={40}
                    unknownCount={50}
                />
            </PageSection>
            <PageSection>
                <p>
                    <span className="console-plugin-template__nice">
                        <CheckCircleIcon /> {'Success!'}
                    </span>{' '}
                    {'Your plugin is working.'}
                </p>
            </PageSection>
        </>
    );
}
