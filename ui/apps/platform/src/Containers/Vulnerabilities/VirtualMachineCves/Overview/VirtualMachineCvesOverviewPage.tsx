import { Card, CardBody, Content, Flex, PageSection, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import TechnologyPreviewLabel from 'Components/PatternFly/PreviewLabel/TechnologyPreviewLabel';

import VirtualMachineScanScopeAlert from '../components/VirtualMachineScanScopeAlert';
import VirtualMachinesCvesTable from './VirtualMachinesCvesTable';

function VirtualMachineCvesOverviewPage() {
    return (
        <>
            <PageTitle title="Virtual Machine CVEs Overview" />
            <PageSection hasBodyWrapper={false} component="div">
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <Title headingLevel="h1">Virtual machine vulnerabilities</Title>
                        <TechnologyPreviewLabel />
                    </Flex>
                    <Content component="p">
                        Prioritize and remediate observed CVEs across virtual machines
                    </Content>
                </Flex>
            </PageSection>
            <PageSection hasBodyWrapper={false} padding={{ default: 'noPadding' }}>
                <PageSection hasBodyWrapper={false} isCenterAligned>
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
