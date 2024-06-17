import React from 'react';
import { Alert, Modal, Text, Button, Flex, Form, Radio, FormGroup } from '@patternfly/react-core';
import { FormikHelpers, useFormik } from 'formik';

import { durations, snoozeDurations } from 'constants/timeWindows';
import useRestMutation from 'hooks/useRestMutation';
import { suppressVulns, unsuppressVulns } from 'services/VulnerabilitiesService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { ValueOf } from 'utils/type.utils';
import { SnoozeAction, SnoozeableCveType } from './useSnoozeCveModal';

const durationOptions = ['DAY', 'WEEK', 'MONTH', 'UNSET'] as const;

type FormValues = {
    cves: string[];
    duration: ValueOf<typeof durations>;
};

export type SnoozeCvesModalProps = {
    action: SnoozeAction;
    cveType: SnoozeableCveType;
    cves: { cve: string }[];
    onSuccess: () => void;
    onClose: () => void;
};

function SnoozeCvesModal({ action, cveType, cves, onSuccess, onClose }: SnoozeCvesModalProps) {
    const { error, mutate, isSuccess, isError } = useRestMutation<FormValues, unknown>(
        ({ cves, duration }) => {
            return action === 'SNOOZE'
                ? suppressVulns(cveType, cves, duration)
                : unsuppressVulns(cveType, cves);
        }
    );

    const { values, setFieldValue, submitForm, isSubmitting } = useFormik({
        initialValues: {
            cves: cves.map(({ cve }) => cve),
            duration: '0',
        },
        onSubmit: (formValues: FormValues, helpers: FormikHelpers<FormValues>) => {
            const callbackOptions = {
                onSuccess,
                onSettled: () => helpers.setSubmitting(false),
            };

            mutate(formValues, callbackOptions);
        },
    });

    const title = action === 'SNOOZE' ? 'Snooze CVEs' : 'Unsnooze CVEs';
    const text =
        action === 'SNOOZE'
            ? 'Snoozed CVEs will not appear in vulnerability reports or trigger policy violations'
            : 'Unsnoozed CVEs will appear in vulnerability reports and trigger policy violations';

    return (
        <Modal
            aria-label={title}
            title={title}
            onClose={onClose}
            isOpen
            variant="small"
            actions={[
                <Button
                    key="perform-modal-action"
                    className="pf-v5-u-display-flex pf-v5-u-align-items-center"
                    isLoading={isSubmitting}
                    isDisabled={isSubmitting || isSuccess}
                    onClick={submitForm}
                    countOptions={{ isRead: true, count: cves.length }}
                >
                    <span>{action === 'SNOOZE' ? 'Snooze CVEs' : 'Unsnooze CVEs'}</span>
                </Button>,
                <Button
                    key="close-modal"
                    isDisabled={isSubmitting}
                    variant="link"
                    onClick={onClose}
                >
                    {isSuccess ? 'Close' : 'Cancel'}
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                {isSuccess && (
                    <Alert variant="success" isInline title="Request submitted successfully" />
                )}
                {isError && (
                    <Alert
                        variant="danger"
                        isInline
                        title="There was an error submitting the request"
                    >
                        {getAxiosErrorMessage(error)}
                    </Alert>
                )}
                <Text>{text}</Text>
                {action === 'SNOOZE' && (
                    <Form style={{ minHeight: 0 }}>
                        <FormGroup fieldId="snooze-duration" label="Snooze duration">
                            <Flex
                                direction={{ default: 'column' }}
                                spaceItems={{ default: 'spaceItemsXs' }}
                            >
                                {durationOptions.map((option) => (
                                    <Radio
                                        id={`snooze-duration-${option}`}
                                        key={option}
                                        isDisabled={isSubmitting || isSuccess}
                                        isChecked={values.duration === durations[option]}
                                        name={option}
                                        onChange={() =>
                                            setFieldValue('duration', durations[option])
                                        }
                                        label={snoozeDurations[option]}
                                    />
                                ))}
                            </Flex>
                        </FormGroup>
                    </Form>
                )}
            </Flex>
        </Modal>
    );
}

export default SnoozeCvesModal;
