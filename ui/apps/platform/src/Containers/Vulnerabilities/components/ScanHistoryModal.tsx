import React, { CSSProperties, useCallback, useState } from 'react';
import {
    Alert,
    Button,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    Modal,
    Text,
    Title,
} from '@patternfly/react-core';
import { ActionsColumn, IAction, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { ExclamationCircleIcon, CheckIcon } from '@patternfly/react-icons';

import Raven from 'raven-js';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useRestMutation from 'hooks/useRestMutation';
import { getImagesScanHistory } from 'services/imageService';
import useRestQuery from 'hooks/useRestQuery';
import { ScanAudit, ScanAuditEvent } from 'types/image.proto';


export type ScanHistoryModalProps = {
    onClose: () => void;
    imageName: string;
};

function ScanHistoryModal(props: ScanHistoryModalProps) {
    const scanHistoryFn = useCallback(() => {
        return getImagesScanHistory(props.imageName);
    },[]);
    const currentScanHistoryRequest = useRestQuery(scanHistoryFn);
    const events = currentScanHistoryRequest.data ?? [];
    console.log(events)

    const { onClose, imageName } = props;
    const [historySubEvents, setHistorySubEvents] = useState<ScanAuditEvent[]>();


    return (
        <Modal
            isOpen
            onClose={onClose}
            variant="large"
            header={
                <Flex
                    className="pf-v5-u-mr-md"
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                >
                    <Title headingLevel="h1">Scan History</Title>
                </Flex>
            }
            actions={[
                <Button key="close-modal" variant="link" onClick={onClose}>
                    Close
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
                <Table variant="compact">
                    <Thead>
                        <Tr>
                            <Th></Th>
                            <Th>Time</Th>
                            <Th>Message</Th>
                            <Th>Details</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {events.map((event, index)=>{
                            var time = event.eventTime.split('.')[0];
                            return (
                                <Tr>
                                    <Td>{
                                            (event.status == "SUCCESS") 
                                                ? <CheckIcon color="var(--pf-v5-global--success-color--100)"  />
                                                : <ExclamationCircleIcon color="var(--pf-v5-global--danger-color--200)"></ExclamationCircleIcon>
                                        }
                                    </Td>
                                    <Td><code className="pf-v5-u-font-family-monospace pf-v5-u-font-size-sm">{time}</code></Td>
                                    <Td modifier="breakWord">{event.message}</Td>
                                    <Td>
                                        <Button variant="link" onClick={() => {
                                            setHistorySubEvents(event.events);
                                        }}>
                                            View
                                        </Button>
                                    </Td>
                                    
                                </Tr>
                            );
                        })}
                    </Tbody>
                </Table>
            </Flex>
            {historySubEvents && (
                <ScanHistorySubModal 
                    onClose={() => setHistorySubEvents(undefined)}
                    events={historySubEvents}
                >
                </ScanHistorySubModal>
            )}
            
        </Modal>
    );
}

export type ScanHistorySubModalProps = {
    onClose: () => void;
    events: ScanAuditEvent[];
};
function ScanHistorySubModal(props: ScanHistorySubModalProps) {
    const { onClose, events } = props;

    return (
        <Modal
            isOpen
            onClose={onClose}
            variant="large"
            header={
                <Flex
                    className="pf-v5-u-mr-md"
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                >
                    <Title headingLevel="h1">Scan Details</Title>
                    
                </Flex>
            }
        >
            <Flex>
                <Table variant="compact">
                    <Thead>
                        <Tr>
                            <Th></Th>
                            <Th>Status</Th>
                            <Th>Message</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {events.map((event, index)=>{
                            var time = event.time.split('.')[0];
                            return (
                                <Tr>
                                    <Td>{
                                            (event.status == "SUCCESS") 
                                                ? <CheckIcon color="var(--pf-v5-global--success-color--100)"  />
                                                : <ExclamationCircleIcon color="var(--pf-v5-global--danger-color--200)"></ExclamationCircleIcon>
                                        }
                                    </Td>
                                    <Td><code className="pf-v5-u-font-family-monospace pf-v5-u-font-size-sm">{time}</code></Td>
                                    <Td modifier="breakWord">{event.message}</Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                </Table>
            </Flex>
        </Modal>
    );
}


export default ScanHistoryModal;
