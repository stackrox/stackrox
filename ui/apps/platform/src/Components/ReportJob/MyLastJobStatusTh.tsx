import React from 'react';
import { Flex, FlexItem } from '@patternfly/react-core';

import HelpIconTh from 'Containers/Vulnerabilities/VulnerablityReporting/VulnReports/HelpIconTh';

function MyLastJobStatusTh() {
    return (
        <HelpIconTh
            popoverContent={
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <FlexItem>
                        <p>
                            The status of your last requested job from the{' '}
                            <strong>active job queue</strong>. An <strong>active job queue</strong>{' '}
                            includes any requested job with the status of <strong>preparing</strong>{' '}
                            or <strong>waiting</strong> until completed.
                        </p>
                    </FlexItem>
                    <FlexItem>
                        <p>
                            <strong>Preparing:</strong>
                        </p>
                        <p>Your last requested job is still being processed.</p>
                    </FlexItem>
                    <FlexItem>
                        <p>
                            <strong>Waiting:</strong>
                        </p>
                        <p>
                            Your last requested job is in the queue and waiting to be processed
                            since other requested jobs are being processed.
                        </p>
                    </FlexItem>
                </Flex>
            }
        >
            My last job status
        </HelpIconTh>
    );
}

export default MyLastJobStatusTh;
