import React, { useState } from 'react';
import { Modal, Button, Alert, Flex, FlexItem } from '@patternfly/react-core';
import { Link } from 'react-router-dom';

import { runViewBasedReport } from 'services/ReportsService';
import { vulnerabilityViewBasedReportsPath } from 'routePaths';
import useAnalytics, { VIEW_BASED_REPORT_GENERATED } from 'hooks/useAnalytics';

type Message =
    | {
          type: 'success';
          value: string;
          reportID: string;
          requestName: string;
      }
    | {
          type: 'error';
          value: string;
      }
    | null;

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
    const { analyticsTrack } = useAnalytics();
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

                // Track successful report request
                const hasFilters = query.trim().length > 0;
                const filterCount = hasFilters ? query.split('&').length : 0;

                analyticsTrack({
                    event: VIEW_BASED_REPORT_GENERATED,
                    properties: {
                        areaOfConcern,
                        hasFilters: hasFilters ? 1 : 0,
                        filterCount,
                    },
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
                    Generate a CSV report based on this view and the filters you’ve applied. Once
                    completed, this report will be available in the view-based reports section until
                    it is purged according to your retention settings.
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
                                    check the status and download the report once it’s ready.
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
