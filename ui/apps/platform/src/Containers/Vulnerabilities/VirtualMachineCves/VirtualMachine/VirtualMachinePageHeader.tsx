import React from 'react';
import { Flex, Title, LabelGroup, Label } from '@patternfly/react-core';

import { getDateTime } from 'utils/dateUtils';
import DeveloperPreviewLabel from 'Components/PatternFly/DeveloperPreviewLabel';

import HeaderLoadingSkeleton from '../../components/HeaderLoadingSkeleton';

// TODO: Move this to the service layer when it's implemented
export type VirtualMachineMetadata = {
    id: string;
    name: string;
    namespace: string;
    description: string;
    status: string;
    ipAddress: string;
    operatingSystem: string;
    guestOS: string;
    agent: string;
    scanTime?: string;
    createdAt?: string;
    owner: string;
    pod: string;
    template: string;
    bootOrder: string[];
    workloadProfile: string;
    cdroms: {
        name: string;
        source: string;
    }[];
    labels: {
        key: string;
        value: string;
    }[];
    annotations: {
        key: string;
        value: string;
    }[];
};

export type VirtualMachinePageHeaderProps = {
    data: VirtualMachineMetadata | undefined;
};

function VirtualMachinePageHeader({ data }: VirtualMachinePageHeaderProps) {
    if (!data) {
        return (
            <HeaderLoadingSkeleton
                nameScreenreaderText="Loading Virtual Machine name"
                metadataScreenreaderText="Loading Virtual Machine metadata"
            />
        );
    }

    return (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                <Title headingLevel="h1">{data.name}</Title>
                <DeveloperPreviewLabel />
            </Flex>
            <LabelGroup numLabels={5}>
                <Label>GuestOS: {data.guestOS}</Label>
                <Label>In: {data.namespace}</Label>
                <Label>Agent: {data.agent}</Label>
                {data.scanTime && <Label>Scan time: {getDateTime(data.scanTime)}</Label>}
                {data.createdAt && <Label>Created: {getDateTime(data.createdAt)}</Label>}
            </LabelGroup>
        </Flex>
    );
}

export default VirtualMachinePageHeader;
