import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    PageSection,
} from '@patternfly/react-core';
import capitalize from 'lodash/capitalize';

import type { VirtualMachine } from 'services/VirtualMachineService';

export type VirtualMachinePageDetailsProps = {
    virtualMachine: VirtualMachine | undefined;
};

function VirtualMachinePageDetails({ virtualMachine }: VirtualMachinePageDetailsProps) {
    const facts = virtualMachine?.facts ?? {};
    return (
        <PageSection hasBodyWrapper={false} isFilled padding={{ default: 'padding' }}>
            <DescriptionList>
                <DescriptionListGroup>
                    <DescriptionListTerm>Status</DescriptionListTerm>
                    <DescriptionListDescription>
                        {capitalize(virtualMachine?.state)}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Operating System</DescriptionListTerm>
                    <DescriptionListDescription>{facts.guestOS || '-'}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>IP Addresses</DescriptionListTerm>
                    <DescriptionListDescription>
                        {facts.ipAddresses || '-'}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Node</DescriptionListTerm>
                    <DescriptionListDescription>{facts.nodeName || '-'}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Pods</DescriptionListTerm>
                    <DescriptionListDescription>
                        {facts.activePods || '-'}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Boot Order</DescriptionListTerm>
                    <DescriptionListDescription>
                        {facts.bootOrder || '-'}
                    </DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
        </PageSection>
    );
}

export default VirtualMachinePageDetails;
