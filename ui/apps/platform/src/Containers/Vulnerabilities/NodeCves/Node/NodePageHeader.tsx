import React from 'react';
import { Flex, Title, LabelGroup, Label } from '@patternfly/react-core';
import { gql } from '@apollo/client';

import { getDateTime } from 'utils/dateUtils';

import HeaderLoadingSkeleton from '../../components/HeaderLoadingSkeleton';

export const nodeMetadataFragment = gql`
    fragment NodeMetadata on Node {
        id
        name
        operatingSystem
        kubeletVersion
        kernelVersion
        scanTime
    }
`;

export type NodeMetadata = {
    id: string;
    name: string;
    operatingSystem: string;
    kubeletVersion: string;
    kernelVersion: string;
    scanTime?: string;
};

export type NodePageHeaderProps = {
    data: NodeMetadata | undefined;
};

function NodePageHeader({ data }: NodePageHeaderProps) {
    if (!data) {
        return (
            <HeaderLoadingSkeleton
                nameScreenreaderText="Loading Node name"
                metadataScreenreaderText="Loading Node metadata"
            />
        );
    }

    const numLabels = data.scanTime ? 4 : 3;

    return (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Title headingLevel="h1" className="pf-u-mb-sm">
                {data.name}
            </Title>
            <LabelGroup numLabels={numLabels}>
                <Label>OS: {data.operatingSystem}</Label>
                <Label>Kubelet: {data.kubeletVersion}</Label>
                <Label>Kernel version: {data.kernelVersion}</Label>
                {data.scanTime && <Label>Scan time: {getDateTime(data.scanTime)}</Label>}
            </LabelGroup>
        </Flex>
    );
}

export default NodePageHeader;
