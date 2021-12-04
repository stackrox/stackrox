import React, { ReactElement } from 'react';
import { Button, Form, Modal, ModalVariant, Radio, TextArea } from '@patternfly/react-core';
import * as yup from 'yup';

import FormLabelGroup from 'Containers/Integrations/IntegrationForm/FormLabelGroup';
import FormMessage, { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import useFormModal from 'hooks/patternfly/useFormModal';

export type UpdateDeferralFormValues = {
    expiresOn: string;
    comment: string;
};

export type UpdateDeferralFormModalProps = {
    isOpen: boolean;
    numRequestsToBeAssessed: number;
    onSendRequest: (values: UpdateDeferralFormValues) => Promise<FormResponseMessage>;
    onCompleteRequest: () => void;
    onCancel: () => void;
};

const EXPIRES_ON = {
    UNTIL_FIXABLE: 'Until Fixable',
    TWO_WEEKS: '2 weeks',
    THIRTY_DAYS: '30 days',
    NINETY_DAYS: '90 days',
    INDEFINITELY: 'Indefinitely',
};

const validationSchema = yup.object().shape({
    expiresOn: yup
        .string()
        .trim()
        .oneOf(Object.values(EXPIRES_ON))
        .required('An expiry time is required'),
    comment: yup.string().trim().required('A deferral rationale is required'),
});

function UpdateDeferralFormModal({
    isOpen,
    numRequestsToBeAssessed,
    onSendRequest,
    onCompleteRequest,
    onCancel,
}: UpdateDeferralFormModalProps): ReactElement {
    // @TODO: Reuse this new hook for the other form modals
    const { formik, message, onHandleSubmit, onHandleCancel } =
        useFormModal<UpdateDeferralFormValues>({
            initialValues: {
                expiresOn: '',
                comment: '',
            },
            validationSchema,
            onCompleteRequest,
            onSendRequest,
            onCancel,
        });

    function onChange(value, event) {
        return formik.setFieldValue(event.target.id, value);
    }

    function onExpiresOnChange(_, event) {
        return formik.setFieldValue('expiresOn', event.target.value);
    }

    const title = `Update deferrals (${numRequestsToBeAssessed})`;

    // @TODO: Create reusable components for the action buttons and form fields
    return (
        <Modal
            variant={ModalVariant.small}
            title={title}
            isOpen={isOpen}
            onClose={onHandleCancel}
            actions={[
                <Button
                    key="confirm"
                    variant="primary"
                    onClick={onHandleSubmit}
                    isDisabled={formik.isSubmitting || !formik.dirty || !formik.isValid}
                    isLoading={formik.isSubmitting}
                >
                    Request update
                </Button>,
                <Button
                    key="cancel"
                    variant="link"
                    onClick={onHandleCancel}
                    isDisabled={formik.isSubmitting}
                >
                    Cancel
                </Button>,
            ]}
        >
            <FormMessage message={message} />
            <div className="pf-u-pb-md">
                The deferral request updates will require another approval by your security lead.
            </div>
            <Form>
                <FormLabelGroup
                    label="How long should the CVEs be deferred?"
                    isRequired
                    fieldId="expiresOn"
                    touched={formik.touched}
                    errors={formik.errors}
                >
                    <Radio
                        id="expiresOnUntilFixable"
                        name="expiresOn"
                        label={EXPIRES_ON.UNTIL_FIXABLE}
                        value={EXPIRES_ON.UNTIL_FIXABLE}
                        isChecked={formik.values.expiresOn === EXPIRES_ON.UNTIL_FIXABLE}
                        onChange={onExpiresOnChange}
                    />
                    <Radio
                        id="expiresOnTwoWeeks"
                        name="expiresOn"
                        label={EXPIRES_ON.TWO_WEEKS}
                        value={EXPIRES_ON.TWO_WEEKS}
                        isChecked={formik.values.expiresOn === EXPIRES_ON.TWO_WEEKS}
                        onChange={onExpiresOnChange}
                    />
                    <Radio
                        id="expiresOnThirtyDays"
                        name="expiresOn"
                        label={EXPIRES_ON.THIRTY_DAYS}
                        value={EXPIRES_ON.THIRTY_DAYS}
                        isChecked={formik.values.expiresOn === EXPIRES_ON.THIRTY_DAYS}
                        onChange={onExpiresOnChange}
                    />
                    <Radio
                        id="expiresOnNinetyDays"
                        name="expiresOn"
                        label={EXPIRES_ON.NINETY_DAYS}
                        value={EXPIRES_ON.NINETY_DAYS}
                        isChecked={formik.values.expiresOn === EXPIRES_ON.NINETY_DAYS}
                        onChange={onExpiresOnChange}
                    />
                    <Radio
                        id="expiresOnIndefinitely"
                        name="expiresOn"
                        label={EXPIRES_ON.INDEFINITELY}
                        value={EXPIRES_ON.INDEFINITELY}
                        isChecked={formik.values.expiresOn === EXPIRES_ON.INDEFINITELY}
                        onChange={onExpiresOnChange}
                    />
                </FormLabelGroup>
                <FormLabelGroup
                    label="Deferral rationale"
                    isRequired
                    fieldId="comment"
                    touched={formik.touched}
                    errors={formik.errors}
                >
                    <TextArea
                        isRequired
                        type="text"
                        id="comment"
                        value={formik.values.comment}
                        onChange={onChange}
                        onBlur={formik.handleBlur}
                    />
                </FormLabelGroup>
            </Form>
        </Modal>
    );
}

export default UpdateDeferralFormModal;
