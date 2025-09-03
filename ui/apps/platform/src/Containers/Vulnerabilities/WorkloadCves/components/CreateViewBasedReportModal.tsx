import React, { useState } from 'react';
import { Modal, Button, Alert, Flex, FlexItem } from '@patternfly/react-core';
import { Link } from 'react-router-dom';

import { runViewBasedReport } from 'services/ReportsService';
import { vulnerabilityViewBasedReportsPath } from 'routePaths';

type Message = {
    type: 'success' | 'error';
    value: string;
    reportID?: string;
    requestName?: string;
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
                    value: 'CSV report generation was triggered.',
                    reportID: response.reportID,
                    requestName: response.requestName,
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
                        title="Report generation started successfully"
                        className="pf-v5-u-w-100"
                        component="p"
                    >
                        {message.reportID && (
                            <Flex
                                direction={{ default: 'column' }}
                                spaceItems={{ default: 'spaceItemsXs' }}
                            >
                                <FlexItem>
                                    <strong>Report Name:</strong>{' '}
                                    {message.requestName || message.reportID}
                                </FlexItem>
                                <FlexItem>
                                    Report generation may take a few minutes to complete. You can
                                    check the status and download the report once it&apos;s ready.
                                </FlexItem>
                                <FlexItem>
                                    <Button
                                        variant="link"
                                        isInline
                                        component={(props) => (
                                            <Link
                                                {...props}
                                                to={vulnerabilityViewBasedReportsPath}
                                            />
                                        )}
                                    >
                                        View status in reports table
                                    </Button>
                                </FlexItem>
                            </Flex>
                        )}
                    </Alert>
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
