import React, { ReactElement, useState } from 'react';
import {
    Button,
    ExpandableSection,
    Flex,
    FlexItem,
    Form,
    PageSection,
    TextArea,
    TextInput,
} from '@patternfly/react-core';
import * as yup from 'yup';
import { FieldArray, FormikProvider } from 'formik';
import { TrashIcon } from '@patternfly/react-icons';

import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormMessage from 'Components/PatternFly/FormMessage';
import { SignatureIntegration } from 'types/signatureIntegration.proto';
import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';
import { IntegrationFormProps } from '../integrationFormTypes';
import useIntegrationForm from '../useIntegrationForm';

const validationSchema = yup.object().shape({
    id: yup.string().trim(),
    name: yup.string().trim().required('Integration name is required'),
    cosign: yup.object().shape({
        publicKeys: yup.array().of(
            yup.object().shape({
                name: yup.string().trim().required('Name is required'),
                publicKeyPemEnc: yup.string().required('Public key value is required'),
            })
        ),
    }),
});

const defaultValues: SignatureIntegration = {
    id: '',
    name: '',
    cosign: {
        publicKeys: [],
    },
};

const VerificationExpandableSection = ({ toggleText, children }): ReactElement => {
    const [isExpanded, setIsExpanded] = useState(false);

    function onToggle() {
        setIsExpanded(!isExpanded);
    }

    return (
        <ExpandableSection
            className="verification-expandable-section"
            toggleText={toggleText}
            onToggle={onToggle}
            isExpanded={isExpanded}
            isIndented
        >
            {children}
        </ExpandableSection>
    );
};

function SignatureIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<SignatureIntegration>): ReactElement {
    const formInitialValues = initialValues
        ? ({ ...defaultValues, ...initialValues } as SignatureIntegration)
        : defaultValues;
    const formik = useIntegrationForm<SignatureIntegration>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const {
        values,
        touched,
        errors,
        dirty,
        isValid,
        setFieldValue,
        handleBlur,
        isSubmitting,
        isTesting,
        onSave,
        onCancel,
        message,
    } = formik;

    function onChange(value, event) {
        setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                <FormMessage message={message} />
                <Form isWidthLimited>
                    <FormikProvider value={formik}>
                        <FormLabelGroup
                            isRequired
                            label="Integration name"
                            fieldId="name"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="text"
                                id="name"
                                value={values.name}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <VerificationExpandableSection toggleText="Cosign">
                            <FieldArray
                                name="cosign.publicKeys"
                                render={(arrayHelpers) => (
                                    <>
                                        {values.cosign.publicKeys.length > 0 &&
                                            values.cosign.publicKeys.map(
                                                (_publicKey, index: number) => (
                                                    <Flex
                                                        // eslint-disable-next-line react/no-array-index-key
                                                        key={`publicKey_${index}`}
                                                        direction={{ default: 'column' }}
                                                    >
                                                        <Flex direction={{ default: 'row' }}>
                                                            <Flex
                                                                direction={{
                                                                    default: 'column',
                                                                }}
                                                                flex={{ default: 'flex_1' }}
                                                            >
                                                                <FlexItem>
                                                                    <FormLabelGroup
                                                                        isRequired
                                                                        label="Public key name"
                                                                        fieldId={`cosign.publicKeys[${index}].name`}
                                                                        touched={touched}
                                                                        errors={errors}
                                                                    >
                                                                        <TextInput
                                                                            isRequired
                                                                            type="text"
                                                                            id={`cosign.publicKeys[${index}].name`}
                                                                            value={
                                                                                values.cosign
                                                                                    .publicKeys[
                                                                                    index
                                                                                ].name || ''
                                                                            }
                                                                            onChange={onChange}
                                                                            onBlur={handleBlur}
                                                                            isDisabled={!isEditable}
                                                                        />
                                                                    </FormLabelGroup>
                                                                </FlexItem>
                                                                <FlexItem
                                                                    spacer={{
                                                                        default:
                                                                            index ===
                                                                            values.cosign.publicKeys
                                                                                .length -
                                                                                1
                                                                                ? 'spacerXs'
                                                                                : 'spacerXl',
                                                                    }}
                                                                >
                                                                    <FormLabelGroup
                                                                        isRequired
                                                                        label="Public key value"
                                                                        fieldId={`cosign.publicKeys[${index}].publicKeyPemEnc`}
                                                                        touched={touched}
                                                                        errors={errors}
                                                                    >
                                                                        <TextArea
                                                                            autoResize
                                                                            resizeOrientation="vertical"
                                                                            isRequired
                                                                            type="text"
                                                                            id={`cosign.publicKeys[${index}].publicKeyPemEnc`}
                                                                            value={
                                                                                values.cosign
                                                                                    .publicKeys[
                                                                                    index
                                                                                ].publicKeyPemEnc ||
                                                                                ''
                                                                            }
                                                                            onChange={onChange}
                                                                            onBlur={handleBlur}
                                                                            isDisabled={!isEditable}
                                                                        />
                                                                    </FormLabelGroup>
                                                                </FlexItem>
                                                            </Flex>
                                                            {isEditable && (
                                                                <FlexItem>
                                                                    <Button
                                                                        variant="plain"
                                                                        aria-label="Delete header key/value pair"
                                                                        style={{
                                                                            transform:
                                                                                'translate(0, 42px)',
                                                                        }}
                                                                        onClick={() =>
                                                                            arrayHelpers.remove(
                                                                                index
                                                                            )
                                                                        }
                                                                    >
                                                                        <TrashIcon />
                                                                    </Button>
                                                                </FlexItem>
                                                            )}
                                                        </Flex>
                                                    </Flex>
                                                )
                                            )}
                                        {isEditable && (
                                            <Flex>
                                                <Button
                                                    variant="link"
                                                    isInline
                                                    onClick={() =>
                                                        arrayHelpers.push({
                                                            name: '',
                                                            publicKeyPemEnc: '',
                                                        })
                                                    }
                                                >
                                                    Add new public key
                                                </Button>
                                            </Flex>
                                        )}
                                    </>
                                )}
                            />
                        </VerificationExpandableSection>
                    </FormikProvider>
                </Form>
            </PageSection>
            {isEditable && (
                <IntegrationFormActions>
                    <FormSaveButton
                        onSave={onSave}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                        isDisabled={!dirty || !isValid}
                    >
                        Save
                    </FormSaveButton>
                    <FormCancelButton onCancel={onCancel}>Cancel</FormCancelButton>
                </IntegrationFormActions>
            )}
        </>
    );
}

export default SignatureIntegrationForm;
