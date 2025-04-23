import React, { ReactElement } from 'react';
import { Button, ModalBoxBody, ModalBoxFooter, Alert } from '@patternfly/react-core';
import { Formik } from 'formik';
import * as yup from 'yup';

import { Policy } from 'types/policy.proto';
import {
    MIN_POLICY_NAME_LENGTH,
    hasDuplicateIdOnly,
    policyOverwriteAllowed,
    checkForBlockedSubmit,
    PolicyImportError,
    PolicyResolution,
} from './PolicyImport.utils';
import DuplicatePolicyForm from './DuplicatePolicyForm';

const RESOLUTION = { resolution: '', newName: '' };

type ImportPolicyJSONErrorProps = {
    handleCancelModal: () => void;
    startImportPolicies: () => void;
    policies: Policy[];
    duplicateErrors: PolicyImportError[];
    errorMessages: string[];
    duplicateResolution: PolicyResolution;
    setDuplicateResolution: (duplicateResolution) => void;
};

function ImportPolicyJSONModalError({
    handleCancelModal,
    startImportPolicies,
    policies,
    duplicateErrors,
    duplicateResolution,
    setDuplicateResolution,
    errorMessages,
}: ImportPolicyJSONErrorProps): ReactElement {
    function updateResolution(key, value) {
        setDuplicateResolution({ ...duplicateResolution, [key]: value });
    }

    const duplicateErrorsOnly = duplicateErrors.length > 0;
    const showKeepBothPolicies = hasDuplicateIdOnly(duplicateErrors);
    const allowOverwriteOption = policyOverwriteAllowed(duplicateErrors);
    const isBlocked = checkForBlockedSubmit({
        numPolicies: policies?.length ?? 0,
        messageType: 'error',
        hasDuplicateErrors: !!duplicateErrors,
        duplicateResolution,
    });

    return (
        <Formik
            initialValues={RESOLUTION}
            onSubmit={() => {}}
            validationSchema={yup.object({
                newName: yup.string().when('resolution', {
                    is: 'rename',
                    then: (newNameSchema) =>
                        newNameSchema
                            .trim()
                            .min(
                                MIN_POLICY_NAME_LENGTH,
                                `A policy name must be at least ${MIN_POLICY_NAME_LENGTH} characters.`
                            ),
                }),
            })}
        >
            <>
                <ModalBoxBody>
                    {duplicateErrorsOnly
                        ? 'Address the errors below to continue importing policies'
                        : 'Correct the errors that are listed, and then try to import again.'}
                    <Alert
                        title={
                            duplicateErrorsOnly
                                ? 'Policy already exists'
                                : 'Policy errors causing import failure'
                        }
                        component="p"
                        variant="danger"
                        className="pf-v5-u-mt-md"
                        isInline
                    >
                        <ul>
                            {errorMessages.map((msg) => (
                                <li key={msg} className="py-2">
                                    {msg}
                                </li>
                            ))}
                        </ul>
                        {duplicateErrorsOnly && (
                            <DuplicatePolicyForm
                                updateResolution={updateResolution}
                                showKeepBothPolicies={showKeepBothPolicies}
                                allowOverwriteOption={allowOverwriteOption}
                            />
                        )}
                    </Alert>
                </ModalBoxBody>
                <ModalBoxFooter>
                    {duplicateErrorsOnly && (
                        <Button
                            key="import"
                            variant="primary"
                            onClick={startImportPolicies}
                            isDisabled={isBlocked}
                        >
                            Resume import
                        </Button>
                    )}
                    <Button key="cancel" variant="link" onClick={handleCancelModal}>
                        Cancel
                    </Button>
                </ModalBoxFooter>
            </>
        </Formik>
    );
}

export default ImportPolicyJSONModalError;
