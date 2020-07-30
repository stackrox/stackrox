import React, { useState, useRef, useCallback } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { useDropzone } from 'react-dropzone';
import { Upload } from 'react-feather';
import pluralize from 'pluralize';
import { Formik } from 'formik';
import * as yup from 'yup';

import CustomDialogue from 'Components/CustomDialogue';
import Message from 'Components/Message';
import { fileUploadColors } from 'constants/visuals/colors';
import { actions as pageActions } from 'reducers/policies/page';
import { importPolicies } from 'services/PoliciesService';
import DuplicatePolicyForm from './DuplicatePolicyForm';
import {
    MIN_POLICY_NAME_LENGTH,
    POLICY_DUPE_ACTIONS,
    parsePolicyImportErrors,
    getResolvedPolicies,
    getErrorMessages,
    hasDuplicateIdOnly,
    checkDupeOnlyErrors,
    checkForBlockedSubmit,
} from './PolicyImport.utils';

const RESOLUTION = { resolution: '', newName: '' };

const PolicyImportDialogue = ({ closeAction, importPolicySuccess }) => {
    const [messageObj, setMessageObj] = useState(null);
    const [policies, setPolicies] = useState([]);
    const [duplicateErrors, setDuplicateErrors] = useState(null);
    const [duplicateResolution, setDuplicateResolution] = useState(RESOLUTION);
    const dialogueRef = useRef(null);

    // trying out Formik for this self-contained form
    // Formik requires an onSubmit handler, but the way our confirm/cancel buttons
    //   our baked into the CustomDialogue component would make plugging into those
    //   buttons have potential far-reaching effects,
    //   so just passing it an empty functional for this use-case
    function noop() {}

    const onDrop = useCallback((acceptedFiles) => {
        setMessageObj(null);
        setDuplicateErrors(null);

        acceptedFiles.forEach((file) => {
            // check file type.
            if (file && !file.name.includes('.json')) {
                setMessageObj({
                    type: 'warn',
                    message: 'Only JSON files are supported.',
                });
                return;
            }

            const reader = new FileReader();
            reader.onload = () => {
                const fileContent = reader.result;
                try {
                    const jsonObj = JSON.parse(fileContent);
                    if (jsonObj?.policies && jsonObj.policies.length > 0) {
                        setPolicies(jsonObj.policies);
                    } else {
                        setMessageObj({
                            type: 'error',
                            message:
                                'The file you selected does not have at least one policy in its policies list.',
                        });
                    }
                } catch (err) {
                    setMessageObj({ type: 'error', message: err.message });
                }
            };
            reader.onerror = (e) => {
                reader.abort();
                setMessageObj({ type: 'error', message: e.message });
            };
            reader.readAsText(file);
        });
    }, []);

    const { getRootProps, getInputProps } = useDropzone({ onDrop });

    function startImport() {
        // Note: this only resolves errors on one policy for MVP,
        //   see decision in comment on Jira story, https://stack-rox.atlassian.net/browse/ROX-4409
        const [policiesToImport, metadata] = getResolvedPolicies(
            policies,
            duplicateErrors,
            duplicateResolution
        );

        importPolicies(policiesToImport, metadata)
            .then((response) => {
                if (response.allSucceeded) {
                    setMessageObj({
                        type: 'info',
                        message: 'Policy successfully imported',
                    });
                    const importedPolicyId = response?.responses[0]?.policy?.id; // API always returns a list, but we only support one policy
                    setTimeout(handleClose, 3000);

                    importPolicySuccess(importedPolicyId);
                } else {
                    const errors = parsePolicyImportErrors(response?.responses);
                    const onlyHasDupeErrors = checkDupeOnlyErrors(errors);
                    if (onlyHasDupeErrors) {
                        setDuplicateErrors(errors[0]);
                    }

                    const errorMessages = getErrorMessages(errors[0]);
                    setMessageObj({
                        type: 'error',
                        message: (
                            <ul>
                                {errorMessages.map((err) => (
                                    <li key={err.type} className="py-2">
                                        {err.msg}
                                    </li>
                                ))}
                            </ul>
                        ),
                    });
                }
            })
            .catch((err) => {
                setMessageObj({
                    type: 'error',
                    message: `A network error occurred: ${err.message}`,
                });
            });
    }

    function updateResolution(key, value) {
        setDuplicateResolution({ ...duplicateResolution, [key]: value });
    }

    function handleClose() {
        closeAction();
    }

    const isBlocked = checkForBlockedSubmit({
        numPolicies: policies?.length || 0,
        messageType: messageObj?.type,
        hasDuplicateErrors: !!duplicateErrors,
        duplicateResolution,
    });
    const showKeepBothPolicies = hasDuplicateIdOnly(duplicateErrors);

    return (
        <CustomDialogue
            className="max-w-3/4 md:max-w-2/3 lg:max-w-1/2 min-w-1/2 md:min-w-1/3"
            title="Import a Policy"
            onConfirm={startImport}
            confirmText="Begin Import"
            confirmDisabled={isBlocked}
            onCancel={handleClose}
        >
            <div
                className="overflow-auto p-4"
                ref={dialogueRef}
                data-testid="policy-import-modal-content"
            >
                <>
                    <div className="flex flex-col bg-base-100 rounded-sm shadow flex-grow flex-shrink-0 mb-4">
                        <div className="my-3 px-3 font-600 text-lg leading-loose text-base-600">
                            Upload a policy that has been exported from StackRox.
                        </div>
                        <div
                            {...getRootProps()}
                            className="bg-warning-100 border border-dashed border-warning-500 cursor-pointer flex flex-col h-full hover:bg-warning-200 justify-center min-h-32 outline-none py-3 self-center uppercase w-full"
                        >
                            <input {...getInputProps()} />
                            <div className="flex flex-shrink-0 flex-col">
                                <div
                                    className="mt-3 h-18 w-18 self-center rounded-full flex items-center justify-center flex-shrink-0"
                                    style={{
                                        background: fileUploadColors.BACKGROUND_COLOR,
                                        color: fileUploadColors.ICON_COLOR,
                                    }}
                                >
                                    <Upload
                                        className="h-8 w-8"
                                        strokeWidth="1.5px"
                                        data-testid="upload-icon"
                                    />
                                </div>
                                <span className="font-700 mt-3 text-center text-warning-800">
                                    Choose a policy file in JSON format
                                </span>
                            </div>
                        </div>
                    </div>
                    {policies?.length > 0 && (
                        <div className="flex flex-col bg-base-100 flex-grow flex-shrink-0 mb-2">
                            <h3 className="b-2 font-700 text-lg">
                                The following {`${pluralize('policy', policies.length)}`} will be
                                imported:
                            </h3>
                            <ul data-testid="policies-to-import">
                                {policies.map((policy) => (
                                    <li
                                        key={policy.id}
                                        className="p-2 text-primary-800 font-600 w-full"
                                    >
                                        {policy.name}
                                    </li>
                                ))}
                            </ul>
                        </div>
                    )}
                    {messageObj && <Message type={messageObj.type} message={messageObj.message} />}
                    {duplicateErrors && (
                        <div className="w-full py-4">
                            <Formik
                                initialValues={RESOLUTION}
                                validationSchema={yup.object({
                                    newName: yup.string().when('resolution', {
                                        is: POLICY_DUPE_ACTIONS.RENAME,
                                        then: yup
                                            .string()
                                            .trim()
                                            .min(
                                                MIN_POLICY_NAME_LENGTH,
                                                `A policy name must be at least ${MIN_POLICY_NAME_LENGTH} characters.`
                                            ),
                                    }),
                                })}
                                onSubmit={noop}
                            >
                                <DuplicatePolicyForm
                                    updateResolution={updateResolution}
                                    showKeepBothPolicies={showKeepBothPolicies}
                                />
                            </Formik>
                        </div>
                    )}
                </>
            </div>
        </CustomDialogue>
    );
};

PolicyImportDialogue.propTypes = {
    closeAction: PropTypes.func.isRequired,
    importPolicySuccess: PropTypes.func.isRequired,
};

const mapDispatchToProps = {
    importPolicySuccess: pageActions.importPolicySuccess,
};

export default connect(null, mapDispatchToProps)(PolicyImportDialogue);
