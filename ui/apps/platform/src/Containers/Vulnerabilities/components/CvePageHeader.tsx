import type { ReactNode } from 'react';
import {
    Content,
    Flex,
    Label,
    LabelGroup,
    List,
    ListItem,
    Title,
    Tooltip,
} from '@patternfly/react-core';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';
import uniqBy from 'lodash/uniqBy';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import useFeatureFlags from 'hooks/useFeatureFlags';
import type { CveBaseInfo } from 'types/cve.proto';
import { getDateTime } from 'utils/dateUtils';

import {
    formatEpssProbabilityAsPercent,
    getCveBaseInfoFromDistroTuples,
} from '../WorkloadCves/Tables/table.utils';
import { getDistroLinkText } from '../utils/textUtils';
import { sortCveDistroList } from '../utils/sortUtils';
import { hasKnownExploit, hasKnownRansomwareCampaignUse } from '../utils/vulnerabilityUtils';
import HeaderLoadingSkeleton from './HeaderLoadingSkeleton';
import KnownExploitLabel from './KnownExploitLabel';
import KnownRansomwareCampaignLabel from './KnownRansomwareCampaignLabel';

export type CveMetadata = {
    cve: string;
    firstDiscoveredInSystem: string | null;
    publishedOn: string | null;
    sourceCount?: number;
    distinctSeverityCount?: number;
    distroTuples: {
        summary: string;
        link: string;
        operatingSystem: string;
        datasource: string;
        cveBaseInfo: CveBaseInfo;
    }[];
};

export type CvePageHeaderProps = {
    data: CveMetadata | undefined;
};

function CvePageHeader({ data }: CvePageHeaderProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isEpssProbabilityColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');

    if (!data) {
        return (
            <HeaderLoadingSkeleton
                nameScreenreaderText="Loading CVE name"
                metadataScreenreaderText="Loading CVE metadata"
            />
        );
    }

    const cveBaseInfo = getCveBaseInfoFromDistroTuples(data.distroTuples);
    const epssProbability = cveBaseInfo?.epss?.epssProbability;
    const hasEpssProbabilityLabel = isEpssProbabilityColumnEnabled && Boolean(cveBaseInfo); // not (yet) for Node CVE

    const labels: ReactNode[] = [];
    if (
        isFeatureFlagEnabled('ROX_SCANNER_V4') &&
        isFeatureFlagEnabled('ROX_CISA_KEV') &&
        hasKnownExploit(cveBaseInfo?.exploit)
    ) {
        labels.push(<KnownExploitLabel key="exploit" isCompact={false} />);
        if (hasKnownRansomwareCampaignUse(cveBaseInfo?.exploit)) {
            labels.push(
                <KnownRansomwareCampaignLabel key="knownRansomwareCampaignUse" isCompact={false} />
            );
        }
    }
    if (hasEpssProbabilityLabel) {
        labels.push(
            <Label key="epssProbability">
                EPSS probability: {formatEpssProbabilityAsPercent(epssProbability)}
            </Label>
        );
    }
    if (data.firstDiscoveredInSystem) {
        labels.push(
            <Label key="firstDiscoveredInSystem">
                First discovered in system: {getDateTime(data.firstDiscoveredInSystem)}
            </Label>,
            <Label key="publishedOn">
                Published: {data.publishedOn ? getDateTime(data.publishedOn) : 'Not available'}
            </Label>
        );
    }
    if (data.sourceCount && data.sourceCount > 1) {
        labels.push(
            <Label key="sourceCount">{data.sourceCount} sources</Label>
        );
    }
    if (data.distinctSeverityCount && data.distinctSeverityCount > 1) {
        labels.push(
            <Tooltip
                key="severityDisagreement"
                content="Sources report different severities for this CVE"
            >
                <Label color="gold" icon={<ExclamationTriangleIcon />}>
                    Severity varies by source
                </Label>
            </Tooltip>
        );
    }

    const prioritizedDistros = uniqBy(sortCveDistroList(data.distroTuples), getDistroLinkText);
    const topDistro = prioritizedDistros[0];

    return (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Title headingLevel="h1" className="pf-v6-u-mb-sm">
                {data.cve}
            </Title>
            {labels.length !== 0 && <LabelGroup numLabels={labels.length}>{labels}</LabelGroup>}
            {topDistro && (
                <>
                    <Content component="p">{topDistro.summary}</Content>
                    <List isPlain>
                        {prioritizedDistros.map((distro) => (
                            <ListItem key={distro.operatingSystem}>
                                <ExternalLink>
                                    <a href={distro.link} target="_blank" rel="noopener noreferrer">
                                        {getDistroLinkText(distro)}
                                    </a>
                                </ExternalLink>
                                {distro.datasource && (
                                    <Label variant="outline" isCompact className="pf-v6-u-ml-sm">
                                        {distro.datasource}
                                    </Label>
                                )}
                            </ListItem>
                        ))}
                    </List>
                </>
            )}
        </Flex>
    );
}

export default CvePageHeader;
