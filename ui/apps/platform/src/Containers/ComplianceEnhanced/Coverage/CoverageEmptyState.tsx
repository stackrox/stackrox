import React from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Bullseye, Flex, FlexItem, PageSection, Text } from '@patternfly/react-core';
import { CubesIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import usePermissions from 'hooks/usePermissions';
import { complianceEnhancedSchedulesPath } from 'routePaths';

import CoveragesPageHeader from './CoveragesPageHeader';

function CoverageEmptyState() {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForCompliance = hasReadWriteAccess('Compliance');
    return (
        <>
            <CoveragesPageHeader />
            <PageSection isFilled>
                <Bullseye className="pf-v5-u-background-color-100">
                    <EmptyStateTemplate
                        title="No scan data available"
                        headingLevel="h2"
                        icon={CubesIcon}
                    >
                        <Flex direction={{ default: 'column' }}>
                            {hasWriteAccessForCompliance && (
                                <FlexItem>
                                    <Text>
                                        Create a scan schedule to assess profile compliance on
                                        selected clusters.
                                    </Text>
                                </FlexItem>
                            )}
                            <FlexItem>
                                <Link to={complianceEnhancedSchedulesPath}>
                                    Go to scan schedules
                                </Link>
                            </FlexItem>
                        </Flex>
                    </EmptyStateTemplate>
                </Bullseye>
            </PageSection>
        </>
    );
}

export default CoverageEmptyState;
