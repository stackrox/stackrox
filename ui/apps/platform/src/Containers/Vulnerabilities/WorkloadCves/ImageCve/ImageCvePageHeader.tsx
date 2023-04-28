import React from 'react';
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
import { ExternalLinkAltIcon } from '@patternfly/react-icons';
import uniqBy from 'lodash/uniqBy';
import { getDateTime } from 'utils/dateUtils';
import { ensureExhaustive } from 'utils/type.utils';
import { graphql } from 'generated/graphql-codegen';
import { ImageCveMetadataFragment } from 'generated/graphql-codegen/graphql';
import { Distro, sortCveDistroList } from '../sortUtils';

export const imageCveMetadataFragment = graphql(/* GraphQL */ `
    fragment ImageCVEMetadata on ImageCVECore {
        cve
        firstDiscoveredInSystem
        distroTuples {
            summary
            link
            operatingSystem
        }
    }
`);

function getDistroLinkText({ distro }: { distro: Distro }): string {
    switch (distro) {
        case 'rhel':
        case 'centos':
            return 'View in Red Hat CVE database';
        case 'ubuntu':
            return 'View in Ubuntu CVE database';
        case 'debian':
            return 'View in Debian CVE database';
        case 'alpine':
            return 'View in Alpine Linux CVE database';
        case 'amzn':
            return 'View in Amazon Linux CVE database';
        case 'other':
            return 'View additional information';
        default:
            return ensureExhaustive(distro);
    }
}

export type ImageCvePageHeaderProps = {
    data: ImageCveMetadataFragment | null | undefined;
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
                                <a
                                    className="pf-u-pl-0"
                                    href={distro.link}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    {getDistroLinkText(distro)}
                                    <ExternalLinkAltIcon className="pf-u-display-inline pf-u-ml-sm" />
                                </a>
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
