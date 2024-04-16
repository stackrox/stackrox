import React from 'react';
import { Flex, LabelGroup, Label, Text, Title, List, ListItem } from '@patternfly/react-core';
import uniqBy from 'lodash/uniqBy';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import { getDateTime } from 'utils/dateUtils';

import { getDistroLinkText } from '../utils/textUtils';
import { sortCveDistroList } from '../utils/sortUtils';
import HeaderLoadingSkeleton from './HeaderLoadingSkeleton';

export type CveMetadata = {
    cve: string;
    firstDiscoveredInSystem: string | null;
    distroTuples: {
        summary: string;
        link: string;
        operatingSystem: string;
    }[];
};

export type CvePageHeaderProps = {
    data: CveMetadata | undefined;
};

function CvePageHeader({ data }: CvePageHeaderProps) {
    if (!data) {
        return (
            <HeaderLoadingSkeleton
                nameScreenreaderText="Loading CVE name"
                metadataScreenreaderText="Loading CVE metadata"
            />
        );
    }

    const prioritizedDistros = uniqBy(sortCveDistroList(data.distroTuples), getDistroLinkText);
    const topDistro = prioritizedDistros[0];

    return (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Title headingLevel="h1" className="pf-v5-u-mb-sm">
                {data.cve}
            </Title>
            {data.firstDiscoveredInSystem && (
                <LabelGroup numLabels={1}>
                    <Label>
                        First discovered in system {getDateTime(data.firstDiscoveredInSystem)}
                    </Label>
                </LabelGroup>
            )}
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
