import { Card, CardBody, Flex, PageSection, Text, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import TechnologyPreviewLabel from 'Components/PatternFly/PreviewLabel/TechnologyPreviewLabel';

import VirtualMachineScanScopeAlert from '../components/VirtualMachineScanScopeAlert';
import VirtualMachinesCvesTable from './VirtualMachinesCvesTable';

function VirtualMachineCvesOverviewPage() {
    return (
        <>
            <PageTitle title="Virtual Machine CVEs Overview" />
            <PageSection component="div" variant="light">
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <Title headingLevel="h1">Virtual machine vulnerabilities</Title>
                        <TechnologyPreviewLabel />
                    </Flex>
                    <Text>Prioritize and remediate observed CVEs across virtual machines</Text>
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody>
                            <VirtualMachineScanScopeAlert />
                            <VirtualMachinesCvesTable />
                        </CardBody>
                    </Card>
                </PageSection>
            </PageSection>
        </>
    );
}

export default VirtualMachineCvesOverviewPage;
