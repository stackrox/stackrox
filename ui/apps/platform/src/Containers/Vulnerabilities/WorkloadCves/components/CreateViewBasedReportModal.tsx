import React, { useState } from 'react';
import { Modal, Button, Alert, Flex, FlexItem } from '@patternfly/react-core';

import { runViewBasedReport } from 'services/ReportsService';

type Message = {
    type: 'success' | 'error';
    value: string;
} | null;

const defaultMessage: Message = null;

export type CreateViewBasedReportModalProps = {
    isOpen: boolean;
    setIsOpen: (value: boolean) => void;
    query: string;
    areaOfConcern: string;
};

function CreateViewBasedReportModal({
    isOpen,
    setIsOpen,
    query,
    areaOfConcern,
}: CreateViewBasedReportModalProps) {
    const [isTriggeringReportGeneration, setIsTriggeringReportGeneration] = useState(false);
    const [message, setMessage] = useState(defaultMessage);

    const handleModalToggle = () => {
        setIsOpen(!isOpen);
        setMessage(null);
    };

    const triggerViewBasedReportGeneration = () => {
        setIsTriggeringReportGeneration(true);

        runViewBasedReport({ query, areaOfConcern })
            .then((response) => {
                setMessage({
                    type: 'success',
                    value: `CSV report generation was triggered. Report ID: ${response.reportID}`,
                });
            })
            .catch((error) => {
                setMessage({
                    type: 'error',
                    value:
                        error?.message ||
                        'Something went wrong. Please contact support for assistance.',
                });
            })
            .finally(() => {
                setIsTriggeringReportGeneration(false);
            });
    };

    return (
        <Modal
            variant="small"
            title="Export report as CSV"
            isOpen={isOpen}
            onClose={handleModalToggle}
            actions={[
                <Button
                    key="confirm"
                    variant="primary"
                    onClick={triggerViewBasedReportGeneration}
                    isLoading={isTriggeringReportGeneration}
                    isDisabled={isTriggeringReportGeneration}
                >
                    Generate report
                </Button>,
            ]}
        >
            <Flex gap={{ default: 'gapMd' }}>
                <FlexItem>
                    Export a view-based CSV report from this view using the filters you&apos;ve
                    applied. Once completed, this report will be available in the one time reports
                    queue until it is purged according to your retention settings.
                </FlexItem>
                {message?.type === 'success' && (
                    <Alert
                        variant="success"
                        isInline
                        title={message.value}
                        className="pf-v5-u-w-100"
                        component="p"
                    />
                )}
                {message?.type === 'error' && (
                    <Alert
                        variant="danger"
                        isInline
                        title={message.value}
                        className="pf-v5-u-w-100"
                        component="p"
                    />
                )}
            </Flex>
        </Modal>
    );
}

export default CreateViewBasedReportModal;
