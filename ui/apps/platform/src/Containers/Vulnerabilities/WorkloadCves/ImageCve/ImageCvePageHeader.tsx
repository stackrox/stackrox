import React from 'react';
import { gql } from '@apollo/client';
import { Button, Flex, LabelGroup, Label, Skeleton, Text, Title } from '@patternfly/react-core';
import { getDateTime } from 'utils/dateUtils';

export type ImageCveMetadata = {
    cve: string;
    firstDiscoveredInSystem: Date | null;
};

export const imageCveMetadataFragment = gql`
    fragment ImageCVEMetadata on ImageCVECore {
        cve
        # TODO summary
        # TODO url
        firstDiscoveredInSystem
    }
`;

export type ImageCvePageHeaderProps = {
    data?: ImageCveMetadata;
};

function ImageCvePageHeader({ data }: ImageCvePageHeaderProps) {
    return data ? (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Title headingLevel="h1" className="pf-u-mb-sm">
                {data.cve}
            </Title>
            <LabelGroup numLabels={1}>
                <Label isCompact>
                    First discovered in system {getDateTime(data.firstDiscoveredInSystem)}
                </Label>
            </LabelGroup>
            <Text>
                Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nullam vel aliquet velit.
                Nullam quis quam ipsum. Suspendisse sit amet consequat mauris. Nam eget neque dolor.
                Fusce ultrices, ante ac lobortis maximus, lacus sapien lobortis nunc, eu euismod
                justo magna in tellus. Sed gravida, nibh ac rhoncus interdum, nunc lacus faucibus
                tortor, in congue arcu est in est. Pellentesque habitant morbi tristique senectus et
                netus et malesuada fames ac turpis egestas. Ut in lorem tellus. Aenean at blandit
                mauris. Phasellus quis mi vitae diam ullamcorper dictum.
            </Text>
            <Button className="pf-u-pl-0" variant="link" href="#TODO">
                View in Red Hat CVE database (TODO)
            </Button>
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
