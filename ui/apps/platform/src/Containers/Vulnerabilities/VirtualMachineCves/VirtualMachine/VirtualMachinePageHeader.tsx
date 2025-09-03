import React from 'react';
import { Flex, Title, LabelGroup, Label } from '@patternfly/react-core';

import { getDateTime } from 'utils/dateUtils';

import HeaderLoadingSkeleton from '../../components/HeaderLoadingSkeleton';

export type VirtualMachineMetadata = {
    id: string;
    name: string;
    guestOS: string;
    location: string;
    agent: string;
    scanTime?: string;
    created?: string;
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
            <Title headingLevel="h1">{data.name}</Title>
            <LabelGroup numLabels={5}>
                <Label>GuestOS: {data.guestOS}</Label>
                <Label>In: {data.location}</Label>
                <Label>Agent: {data.agent}</Label>
                {data.scanTime && <Label>Scan time: {getDateTime(data.scanTime)}</Label>}
                {data.created && <Label>Created: {getDateTime(data.created)}</Label>}
            </LabelGroup>
        </Flex>
    );
}

export default VirtualMachinePageHeader;
