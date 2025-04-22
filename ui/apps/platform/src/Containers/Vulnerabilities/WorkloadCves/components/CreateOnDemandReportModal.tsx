import React, { useState } from 'react';
import { Modal, Button, Alert, Flex, FlexItem } from '@patternfly/react-core';

type Message = {
    type: 'success' | 'error';
    value: string;
} | null;

const defaultMessage: Message = null;

export type CreateOnDemandReportModalProps = {
    isOpen: boolean;
    setIsOpen: (value: boolean) => void;
    query: string;
    areaOfConcern: string;
};

function CreateOnDemandReportModal({
    isOpen,
    setIsOpen,
    // @TODO: Will use "query" in a future PR
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    query,
    // @TODO: Will use "areaOfConcern" in a future PR
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    areaOfConcern,
}: CreateOnDemandReportModalProps) {
    const [isTriggeringReportGeneration, setIsTriggeringReportGeneration] = useState(false);
    const [message, setMessage] = useState(defaultMessage);

    const handleModalToggle = () => {
        setIsOpen(!isOpen);
        setMessage(null);
    };

    const triggerOnDemandReportGeneration = () => {
        setIsTriggeringReportGeneration(true);
        // @TODO: Do an actual API call. This is just for demonstration.
        const promise = new Promise((resolve, reject) => {
            setTimeout(() => {
                if (Math.random() < 0.5) {
                    resolve({ reportID: '123456789' });
                } else {
                    reject('Something went wrong. Please contact support for assistance.');
                }
            }, 2000);
        });
        promise
            .then(() => {
                // @TODO: Render link to the on-demand reports table
                setMessage({
                    type: 'success',
                    value: 'CSV report generation was triggered.',
                });
            })
            .catch((error) => {
                setMessage({
                    type: 'error',
                    value: error,
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
                    onClick={triggerOnDemandReportGeneration}
                    isLoading={isTriggeringReportGeneration}
                    isDisabled={isTriggeringReportGeneration}
                >
                    Generate report
                </Button>,
            ]}
        >
            <Flex gap={{ default: 'gapMd' }}>
                <FlexItem>
                    Export an on-demand CSV report from this view using the filters you&apos;ve
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

export default CreateOnDemandReportModal;
