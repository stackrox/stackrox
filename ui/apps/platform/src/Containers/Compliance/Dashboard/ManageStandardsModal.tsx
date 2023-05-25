import React, { ReactElement, useState } from 'react';
import { Alert, Button, Checkbox, Form, Modal } from '@patternfly/react-core';
import { useFormik } from 'formik';

import {
    ComplianceStandardMetadata,
    fetchComplianceStandardsSortedByName,
    patchComplianceStandard,
} from 'services/ComplianceService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type ManageStandardsModalProps = {
    standards: ComplianceStandardMetadata[];
    onCancel: () => void;
    onSave: (standards: ComplianceStandardMetadata[]) => void;
};

/*
 * Formik values have standard id as key and showScanResults as value.
 * Therefore, negate hideScanResults value here and
 * showComplianceScanMap[id] value in onSubmit function below.
 */
function getShowScanResultsMap(standards: ComplianceStandardMetadata[]): Record<string, boolean> {
    return Object.fromEntries(standards.map(({ id, hideScanResults }) => [id, !hideScanResults]));
}

function ManageStandardsModal({ standards, onSave, onCancel }): ReactElement {
    const [errorMessage, setErrorMessage] = useState('');
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

            // TODO rewrite with ES2020 allSettled to solve async problem with all.
            // TODO decide how to display results and update Formik state
            // if some requests fail but other requests succeed.
            Promise.all(patchRequestPromises)
                .then(() => {
                    fetchComplianceStandardsSortedByName()
                        .then((standardsFetchedAfterPatchRequests) => {
                            onSave(standardsFetchedAfterPatchRequests);
                        })
                        .catch((error) => {
                            setErrorMessage(getAxiosErrorMessage(error));
                        })
                        .finally(() => {
                            setSubmitting(false);
                        });
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
                <Button key="Cancel" variant="link" onClick={onCancel} isDisabled={isSubmitting}>
                    Cancel
                </Button>,
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
                    variant="warning"
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
