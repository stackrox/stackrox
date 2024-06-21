import React, { useContext } from 'react';
import { Button, Divider, Flex, FlexItem, PageSection, Text, Title } from '@patternfly/react-core';
import { TimesCircleIcon } from '@patternfly/react-icons';

import ScanConfigurationSelect from './components/ScanConfigurationSelect';
import { ScanConfigurationsContext } from './ScanConfigurationsProvider';

function CoveragesPageHeader() {
    const { setSelectedScanConfigName } = useContext(ScanConfigurationsContext);
    return (
        <>
            <PageSection component="div" variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
                    <FlexItem className="pf-v5-u-p-lg">
                        <Title headingLevel="h1">Coverage</Title>
                        <Text>
                            Assess profile compliance for nodes and platform resources across
                            clusters
                        </Text>
                    </FlexItem>
                    <Divider />
                    <Flex
                        className="pf-v5-u-px-lg pf-v5-u-py-sm"
                        justifyContent={{ default: 'justifyContentSpaceBetween' }}
                    >
                        <ScanConfigurationSelect />
                        <Button
                            variant="link"
                            icon={<TimesCircleIcon />}
                            onClick={() => setSelectedScanConfigName(undefined)}
                        >
                            Reset filter
                        </Button>
                    </Flex>
                </Flex>
            </PageSection>
        </>
    );
}

export default CoveragesPageHeader;
