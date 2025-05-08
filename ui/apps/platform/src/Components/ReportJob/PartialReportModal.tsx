import React from 'react';
import { Button, Flex, FlexItem, Modal } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import sortBy from 'lodash/sortBy';

import { FailedCluster } from 'types/reportJob';
import {isUndefined} from "lodash";

export type PartialReportModalProps = {
    failedClusters?: FailedCluster[];
    onConfirm?: () => void;
};

function PartialReportModal({ failedClusters = [], onConfirm}: PartialReportModalProps) {
    const [isModalOpen, setIsModalOpen] = React.useState(false);

    const handleModalToggle = () => {
        setIsModalOpen(!isModalOpen);
    };

    const sortedFailedClusters = sortBy(failedClusters, 'clusterName');

    const buttonText = isUndefined(onConfirm) ? "Partial report sent" : "Partial report"
    const actions = isUndefined(onConfirm) ? [
        <Button key="cancel" variant="link" onClick={handleModalToggle}>
        Cancel
        </Button>,
    ] : [
        <Button
            key="confirm"
            variant="primary"
            onClick={() => {
                handleModalToggle();
                onConfirm();
            }}
        >
            Download partial report
        </Button>,
        <Button key="cancel" variant="link" onClick={handleModalToggle}>
            Cancel
        </Button>,
    ]

    return (
        <React.Fragment>
            <Button
                variant="link"
                isInline
                className="pf-v5-u-primary-color-100"
                onClick={handleModalToggle}
            >
                {buttonText}
            </Button>
            <Modal
                variant="medium"
                title="Partial report generated"
                isOpen={isModalOpen}
                onClose={handleModalToggle}
                actions={{actions}}
            >
                <Flex>
                    <FlexItem>
                        The scan on the listed clusters could not be scheduled, and the report could
                        not be generated. Please check the cluster logs to diagnose the issue.
                    </FlexItem>
                    <Table aria-label="Failed clusters table" variant="compact">
                        <Thead>
                            <Tr>
                                <Th width={30}>Cluster</Th>
                                <Th width={50}>Reason</Th>
                                <Th width={20}>Operator version</Th>
                            </Tr>
                        </Thead>
                        <Tbody>
                            {sortedFailedClusters.map((cluster) => (
                                <Tr key={cluster.clusterId}>
                                    <Td dataLabel="Cluster">{cluster.clusterName}</Td>
                                    <Td dataLabel="Reason">{cluster.reason}</Td>
                                    <Td dataLabel="Operator version">{cluster.operatorVersion}</Td>
                                </Tr>
                            ))}
                        </Tbody>
                    </Table>
                </Flex>
            </Modal>
        </React.Fragment>
    );
}

export default PartialReportModal;
