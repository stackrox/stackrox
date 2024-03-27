import React from 'react';
import { gql } from '@apollo/client';
import {
    Flex,
    LabelGroup,
    Label,
    Skeleton,
    Text,
    Title,
    List,
    ListItem,
} from '@patternfly/react-core';
import uniqBy from 'lodash/uniqBy';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import { getDateTime } from 'utils/dateUtils';

import { getDistroLinkText } from '../../utils/textUtils';
import { sortCveDistroList } from '../../utils/sortUtils';

export type ImageCveMetadata = {
    cve: string;
    firstDiscoveredInSystem: string | null;
    distroTuples: {
        summary: string;
        link: string;
        operatingSystem: string;
    }[];
};

export const imageCveMetadataFragment = gql`
    fragment ImageCVEMetadata on ImageCVECore {
        cve
        firstDiscoveredInSystem
        distroTuples {
            summary
            link
            operatingSystem
        }
    }
`;

export type ImageCvePageHeaderProps = {
    data?: ImageCveMetadata;
};

function ImageCvePageHeader({ data }: ImageCvePageHeaderProps) {
    const prioritizedDistros = uniqBy(
        sortCveDistroList(data?.distroTuples ?? []),
        getDistroLinkText
    );
    return data ? (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Title headingLevel="h1" className="pf-u-mb-sm">
                {data.cve}
            </Title>
            <LabelGroup numLabels={1}>
                {data.firstDiscoveredInSystem && (
                    <Label>
                        First discovered in system {getDateTime(data.firstDiscoveredInSystem)}
                    </Label>
                )}
            </LabelGroup>
            {prioritizedDistros.length > 0 && (
                <>
                    <Text>{prioritizedDistros[0].summary}</Text>
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
    ) : (
        <Flex
            direction={{ default: 'column' }}
            spaceItems={{ default: 'spaceItemsXs' }}
            className="pf-u-w-50"
        >
            <Skeleton screenreaderText="Loading CVE name" fontSize="2xl" />
            <Skeleton screenreaderText="Loading CVE metadata" fontSize="sm" />
        </Flex>
    );
}

export default ImageCvePageHeader;
