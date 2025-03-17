import React, { useState } from 'react';
import { Modal, Button, Alert, Flex, FlexItem } from '@patternfly/react-core';

function ModalBasic({ isOpen, setIsOpen }) {
    const [isGeneratingReport, setIsGeneratingReport] = useState(false);
    const [errorMessage, setErrorMessage] = useState('');
    const [successMessage, setSuccessMessage] = useState('');

    const handleModalToggle = (_event: KeyboardEvent | React.MouseEvent) => {
        setIsOpen(!isOpen);
        setErrorMessage('');
        setSuccessMessage('');
    };

    const handleAdHocReportGeneration = () => {
        setIsGeneratingReport(true);
        const promise = new Promise((resolve, reject) => {
            setTimeout(() => {
                resolve({ reportID: '123456789' });
                // reject('Something went wrong. Please contact support for assistance.');
            }, 2000);
        });
        promise
            .then(() => {
                setIsGeneratingReport(false);
                setSuccessMessage('CSV report generation was triggered.');
                setErrorMessage('');
            })
            .catch((error) => {
                setIsGeneratingReport(false);
                setErrorMessage(error);
                setSuccessMessage('');
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
                    onClick={handleAdHocReportGeneration}
                    isLoading={isGeneratingReport}
                    isDisabled={isGeneratingReport}
                >
                    Generate report
                </Button>,
            ]}
        >
            <Flex gap={{ default: 'gapMd' }}>
                <FlexItem>
                    Export a one time CSV report from this view using the filters you've applied.
                    Once completed, this report will be available in the one time reports queue
                    until it is purged according to your retention settings.
                </FlexItem>
                {errorMessage && (
                    <Alert
                        variant="danger"
                        isInline
                        title={errorMessage}
                        className="pf-v5-u-w-100"
                    />
                )}
                {successMessage && (
                    <Alert
                        variant="success"
                        isInline
                        title={successMessage}
                        className="pf-v5-u-w-100"
                    />
                )}
            </Flex>
        </Modal>
    );
}

export default ModalBasic;
