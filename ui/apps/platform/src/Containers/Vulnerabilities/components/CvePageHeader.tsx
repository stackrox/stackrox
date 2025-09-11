import React from 'react';
import type { ReactNode } from 'react';
import { Flex, LabelGroup, Label, Text, Title, List, ListItem } from '@patternfly/react-core';
import uniqBy from 'lodash/uniqBy';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { CveBaseInfo } from 'types/cve.proto';
import { getDateTime } from 'utils/dateUtils';

import {
    formatEpssProbabilityAsPercent,
    getCveBaseInfoFromDistroTuples,
} from '../WorkloadCves/Tables/table.utils';
import { getDistroLinkText } from '../utils/textUtils';
import { sortCveDistroList } from '../utils/sortUtils';
import HeaderLoadingSkeleton from './HeaderLoadingSkeleton';
// import KnownExploitLabel from './KnownExploitLabel';

export type CveMetadata = {
    cve: string;
    firstDiscoveredInSystem: string | null;
    publishedOn: string | null;
    distroTuples: {
        summary: string;
        link: string;
        operatingSystem: string;
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
    /*
    // Ross CISA KEV
    // TODO replace key prop value with property name
    if (isFeatureFlagEnabled('ROX_SCANNER_V4') && isFeatureFlagEnabled('ROX_WHATEVER') && TODO) {
        labels.push(<KnownExploitLabel key="knownExploit" isCompact={false} />);
        // Future code if design decision is separate labels.
        // if (TODO) {
        //     labels.push(
        //         <KnownExploitLabel
        //             key="knownRansomware"
        //             isCompact={false}
        //             isKnownToBeUsedInRansomwareCampaigns
        //         />
        //     );
        }
    }
    */
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

    const prioritizedDistros = uniqBy(sortCveDistroList(data.distroTuples), getDistroLinkText);
    const topDistro = prioritizedDistros[0];

    return (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Title headingLevel="h1" className="pf-v5-u-mb-sm">
                {data.cve}
            </Title>
            {labels.length !== 0 && <LabelGroup numLabels={labels.length}>{labels}</LabelGroup>}
            {topDistro && (
                <>
                    <Text>{topDistro.summary}</Text>
                    <List isPlain>
                        {prioritizedDistros.map((distro) => (
                            <ListItem key={distro.operatingSystem}>
                                <ExternalLink>
                                    <a href={distro.link} target="_blank" rel="noopener noreferrer">
                                        {getDistroLinkText(distro)}
                                    </a>
                                </ExternalLink>
                            </ListItem>
                        ))}
                    </List>
                </>
            )}
        </Flex>
    );
}

export default CvePageHeader;
