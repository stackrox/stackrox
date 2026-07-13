import { Content, Flex, Label, LabelGroup, Title } from '@patternfly/react-core';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import { getDateTime } from 'utils/dateUtils';

import type { VMCVEDetail } from 'services/VirtualMachineService';
import { formatEpssProbabilityAsPercent } from '../../WorkloadCves/Tables/table.utils';
import HeaderLoadingSkeleton from '../../components/HeaderLoadingSkeleton';

type VirtualMachineCvePageHeaderProps = {
    cveDetail: VMCVEDetail | undefined;
};

function VirtualMachineCvePageHeader({ cveDetail }: VirtualMachineCvePageHeaderProps) {
    if (!cveDetail) {
        return (
            <HeaderLoadingSkeleton
                nameScreenreaderText="Loading CVE name"
                metadataScreenreaderText="Loading CVE metadata"
            />
        );
    }

    return (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Title headingLevel="h1">{cveDetail.cve}</Title>
            <LabelGroup numLabels={3}>
                <Label>
                    EPSS probability: {formatEpssProbabilityAsPercent(cveDetail.epssProbability)}
                </Label>
                <Label>First discovered in system: {getDateTime(cveDetail.firstDiscovered)}</Label>
                <Label>Published: {getDateTime(cveDetail.publishedOn)}</Label>
            </LabelGroup>
            {cveDetail.summary && <Content component="p">{cveDetail.summary}</Content>}
            {cveDetail.link && (
                <ExternalLink>
                    <a href={cveDetail.link} target="_blank" rel="noopener noreferrer">
                        View in Red Hat CVE database
                    </a>
                </ExternalLink>
            )}
        </Flex>
    );
}

export default VirtualMachineCvePageHeader;
