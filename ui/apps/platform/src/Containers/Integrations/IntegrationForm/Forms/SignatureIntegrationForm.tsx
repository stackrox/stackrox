import React, { ReactElement, useState } from 'react';
import {
    Button,
    ExpandableSection,
    Flex,
    FlexItem,
    Form,
    PageSection,
    Popover,
    TextArea,
    TextInput,
} from '@patternfly/react-core';
import * as yup from 'yup';
import { FieldArray, FormikProvider } from 'formik';
import { HelpIcon, TrashIcon } from '@patternfly/react-icons';

import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormMessage from 'Components/PatternFly/FormMessage';
import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import PopoverBodyContent from 'Components/PopoverBodyContent';
import {
    CosignCertificateVerification,
    CosignPublicKey,
    SignatureIntegration,
} from 'types/signatureIntegration.proto';
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
    cosignCertificates: yup.array().of(
        yup.object().shape({
            certificateChainPemEnc: yup.string(),
            certificatePemEnc: yup.string(),
            certificateOidcIssuer: yup
                .string()
                .trim()
                .required('Certificate OIDC issuer is required'),
            certificateIdentity: yup.string().trim().required('Certificate identity is required'),
        })
    ),
});

const defaultValues: SignatureIntegration = {
    id: '',
    name: '',
    cosign: {
        publicKeys: [],
    },
    cosignCertificates: [],
};

const defaultValuesOfCosignCertificateVerification: CosignCertificateVerification = {
    certificateChainPemEnc: '',
    certificatePemEnc: '',
    certificateOidcIssuer: '',
    certificateIdentity: '',
};

const defaultValuesOfCosignPublicKeys: CosignPublicKey = {
    name: '',
    publicKeyPemEnc: '',
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

function regularExpressionIcon(): ReactElement {
    return (
        <Popover
            aria-label="Supports regular expressions"
            bodyContent={
                <PopoverBodyContent
                    headerContent="Supports regular expressions"
                    bodyContent={
                        <ExternalLink>
                            <a
                                href="https://golang.org/s/re2syntax"
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                See RE2 syntax reference
                            </a>
                        </ExternalLink>
                    }
                />
            }
        >
            <button
                type="button"
                aria-label="More info for input"
                onClick={(e) => e.preventDefault()}
                aria-describedby="simple-form-name-01"
                className="pf-v5-c-form__group-label-help"
            >
                <HelpIcon />
            </button>
        </Popover>
    );
}

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
                                onChange={(event, value) => onChange(value, event)}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <VerificationExpandableSection toggleText="Cosign public keys">
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
                                                                            onChange={(
                                                                                event,
                                                                                value
                                                                            ) =>
                                                                                onChange(
                                                                                    value,
                                                                                    event
                                                                                )
                                                                            }
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
                                                                            onChange={(
                                                                                event,
                                                                                value
                                                                            ) =>
                                                                                onChange(
                                                                                    value,
                                                                                    event
                                                                                )
                                                                            }
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
                                                                        aria-label="Delete public key"
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
                                                        arrayHelpers.push(
                                                            defaultValuesOfCosignPublicKeys
                                                        )
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
                        <VerificationExpandableSection toggleText="Cosign certificates">
                            <FieldArray
                                name="cosignCertificates"
                                render={(arrayHelpers) => (
                                    <>
                                        {values.cosignCertificates.length > 0 &&
                                            values.cosignCertificates.map(
                                                (_certificate, index: number) => (
                                                    <Flex
                                                        // eslint-disable-next-line react/no-array-index-key
                                                        key={`certificate_${index}`}
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
                                                                        label="Certificate OIDC issuer"
                                                                        labelIcon={regularExpressionIcon()}
                                                                        fieldId={`cosignCertificates[${index}].certificateOidcIssuer`}
                                                                        touched={touched}
                                                                        errors={errors}
                                                                    >
                                                                        <TextInput
                                                                            isRequired
                                                                            type="text"
                                                                            id={`cosignCertificates[${index}].certificateOidcIssuer`}
                                                                            value={
                                                                                values
                                                                                    .cosignCertificates[
                                                                                    index
                                                                                ]
                                                                                    .certificateOidcIssuer ||
                                                                                ''
                                                                            }
                                                                            onChange={(
                                                                                event,
                                                                                value
                                                                            ) =>
                                                                                onChange(
                                                                                    value,
                                                                                    event
                                                                                )
                                                                            }
                                                                            onBlur={handleBlur}
                                                                            isDisabled={!isEditable}
                                                                        />
                                                                    </FormLabelGroup>
                                                                    <FormLabelGroup
                                                                        isRequired
                                                                        label="Certificate identity"
                                                                        labelIcon={regularExpressionIcon()}
                                                                        fieldId={`cosignCertificates[${index}].certificateIdentity`}
                                                                        touched={touched}
                                                                        errors={errors}
                                                                    >
                                                                        <TextInput
                                                                            isRequired
                                                                            type="text"
                                                                            id={`cosignCertificates[${index}].certificateIdentity`}
                                                                            value={
                                                                                values
                                                                                    .cosignCertificates[
                                                                                    index
                                                                                ]
                                                                                    .certificateIdentity ||
                                                                                ''
                                                                            }
                                                                            onChange={(
                                                                                event,
                                                                                value
                                                                            ) =>
                                                                                onChange(
                                                                                    value,
                                                                                    event
                                                                                )
                                                                            }
                                                                            onBlur={handleBlur}
                                                                            isDisabled={!isEditable}
                                                                        />
                                                                    </FormLabelGroup>
                                                                </FlexItem>
                                                                <FlexItem
                                                                    spacer={{
                                                                        default:
                                                                            index ===
                                                                            values
                                                                                .cosignCertificates
                                                                                .length -
                                                                                1
                                                                                ? 'spacerXs'
                                                                                : 'spacerXl',
                                                                    }}
                                                                >
                                                                    <FormLabelGroup
                                                                        label="Certificate Chain PEM encoded"
                                                                        fieldId={`cosignCertificates[${index}].certificateChainPemEnc`}
                                                                        touched={touched}
                                                                        errors={errors}
                                                                    >
                                                                        <TextArea
                                                                            autoResize
                                                                            resizeOrientation="vertical"
                                                                            isRequired
                                                                            type="text"
                                                                            id={`cosignCertificates[${index}].certificateChainPemEnc`}
                                                                            value={
                                                                                values
                                                                                    .cosignCertificates[
                                                                                    index
                                                                                ]
                                                                                    .certificateChainPemEnc ||
                                                                                ''
                                                                            }
                                                                            onChange={(
                                                                                event,
                                                                                value
                                                                            ) =>
                                                                                onChange(
                                                                                    value,
                                                                                    event
                                                                                )
                                                                            }
                                                                            onBlur={handleBlur}
                                                                            isDisabled={!isEditable}
                                                                        />
                                                                    </FormLabelGroup>
                                                                    <FormLabelGroup
                                                                        label="Certificate PEM encoded"
                                                                        fieldId={`cosignCertificates[${index}].certificatePemEnc`}
                                                                        touched={touched}
                                                                        errors={errors}
                                                                    >
                                                                        <TextArea
                                                                            autoResize
                                                                            resizeOrientation="vertical"
                                                                            type="text"
                                                                            id={`cosignCertificates[${index}].certificatePemEnc`}
                                                                            value={
                                                                                values
                                                                                    .cosignCertificates[
                                                                                    index
                                                                                ]
                                                                                    .certificatePemEnc ||
                                                                                ''
                                                                            }
                                                                            onChange={(
                                                                                event,
                                                                                value
                                                                            ) =>
                                                                                onChange(
                                                                                    value,
                                                                                    event
                                                                                )
                                                                            }
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
                                                                        aria-label="Delete certificate verification data"
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
                                                        arrayHelpers.push(
                                                            defaultValuesOfCosignCertificateVerification
                                                        )
                                                    }
                                                >
                                                    Add new certificate verification
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
