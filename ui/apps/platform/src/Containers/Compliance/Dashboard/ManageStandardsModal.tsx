import React, { ReactElement, useState } from 'react';
import { Alert, Button, Checkbox, Form, Modal } from '@patternfly/react-core';
import { useFormik } from 'formik';

import { ComplianceStandardMetadata, patchComplianceStandard } from 'services/ComplianceService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type ManageStandardsModalProps = {
    onCancel: () => void;
    onChange: () => void;
    standards: ComplianceStandardMetadata[];
};

/*
 * Formik values have standard id as key and showScanResults as value.
 * Therefore, negate hideScanResults value here and
 * showComplianceScanMap[id] value in onSubmit function below.
 */
function getShowScanResultsMap(standards: ComplianceStandardMetadata[]): Record<string, boolean> {
    return Object.fromEntries(standards.map(({ id, hideScanResults }) => [id, !hideScanResults]));
}

function ManageStandardsModal({
    onCancel,
    onChange,
    standards,
}: ManageStandardsModalProps): ReactElement {
    const [errorMessage, setErrorMessage] = useState('');
    const [countFulfilledWhenRejected, setCountFulfilledWhenRejected] = useState(0);
    const { dirty, handleSubmit, isSubmitting, setFieldValue, setSubmitting, values } = useFormik<
        Record<string, boolean>
    >({
        initialValues: getShowScanResultsMap(standards),
        onSubmit: (showScanResultsMap) => {
            setErrorMessage('');
            // Filter standards for which hideScanResults property has changed,
            // and them map to promises for patch requests.
            // Negate hideScanResults is correct even if property is absent.
            const patchRequestPromises = standards
                .filter(({ hideScanResults, id }) => !hideScanResults !== showScanResultsMap[id])
                .map(({ id }) => patchComplianceStandard(id, !showScanResultsMap[id]));

            Promise.allSettled(patchRequestPromises)
                .then((results) => {
                    let numberOfRejectedPromises = 0;
                    let reasonMessage = '';
                    results.forEach((result) => {
                        if (result.status === 'rejected') {
                            if (numberOfRejectedPromises === 0) {
                                reasonMessage = getAxiosErrorMessage(result.reason);
                            }
                            numberOfRejectedPromises += 1;
                        }
                    });
                    setSubmitting(false);

                    if (numberOfRejectedPromises === 0) {
                        onChange();
                    } else {
                        setErrorMessage(reasonMessage);
                        setCountFulfilledWhenRejected(results.length - numberOfRejectedPromises);
                    }
                })
                .catch((error) => {
                    setErrorMessage(getAxiosErrorMessage(error));
                    setSubmitting(false);
                });
        },
    });

    return (
        <Modal
            title="Manage standards"
            variant="small"
            isOpen
            showClose={false}
            actions={[
                <Button
                    key="Save"
                    variant="primary"
                    onClick={() => {
                        handleSubmit();
                    }}
                    isDisabled={!dirty || isSubmitting}
                    isLoading={isSubmitting}
                >
                    Save
                </Button>,
                countFulfilledWhenRejected === 0 ? (
                    <Button
                        key="Cancel"
                        variant="secondary"
                        onClick={onCancel}
                        isDisabled={isSubmitting}
                    >
                        Cancel
                    </Button>
                ) : (
                    <Button
                        key="Close"
                        variant="secondary"
                        onClick={onChange}
                        isDisabled={isSubmitting}
                    >
                        Close
                    </Button>
                ),
            ]}
        >
            <Form>
                {standards.map(({ id, name }) => {
                    return (
                        <Checkbox
                            key={id}
                            id={id}
                            name={id}
                            label={name}
                            isChecked={values[id]}
                            onChange={(value) => {
                                return setFieldValue(id, value);
                            }}
                        />
                    );
                })}
            </Form>
            {errorMessage && (
                <Alert
                    title="Unable to save changes"
                    variant="danger"
                    isInline
                    className="pf-u-mt-lg"
                >
                    {errorMessage}
                </Alert>
            )}
        </Modal>
    );
}

export default ManageStandardsModal;
