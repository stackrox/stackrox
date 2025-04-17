import React, { ReactElement, useState } from 'react';
import {
    Alert,
    Button,
    Checkbox,
    ExpandableSection,
    Flex,
    FlexItem,
    Form,
    PageSection,
    Text,
    TextArea,
    TextInput,
} from '@patternfly/react-core';
import * as yup from 'yup';
import { FieldArray, FormikProvider } from 'formik';
import { TrashIcon } from '@patternfly/react-icons';
import merge from 'lodash/merge';

import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormMessage from 'Components/PatternFly/FormMessage';
import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import {
    CertificateTransparencyLogVerification,
    CosignCertificateVerification,
    CosignPublicKey,
    SignatureIntegration,
    TransparencyLogVerification,
} from 'types/signatureIntegration.proto';
import useMetadata from 'hooks/useMetadata';
import { getVersionedDocs } from 'utils/versioning';
import IntegrationFormActions from '../IntegrationFormActions';
import IntegrationHelpIcon from './Components/IntegrationHelpIcon';
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
            certificateTransparencyLog: yup.object().shape({
                enabled: yup.boolean(),
                publicKeyPemEnc: yup.string().trim(),
            }),
        })
    ),
    transparencyLog: yup.object().shape({
        enabled: yup.boolean(),
        publicKeyPemEnc: yup.string().trim(),
        url: yup.string().trim(),
        validateOffline: yup.boolean(),
    }),
});

// Default values for newly created integrations.
const defaultValues: SignatureIntegration = {
    id: '',
    name: '',
    cosign: {
        publicKeys: [],
    },
    cosignCertificates: [],
    transparencyLog: {
        enabled: true,
        publicKeyPemEnc: '',
        url: 'https://rekor.sigstore.dev',
        validateOffline: false,
    },
};

// Default values for newly created integrations.
const defaultValuesOfCosignCertificateVerification: CosignCertificateVerification = {
    certificateChainPemEnc: '',
    certificatePemEnc: '',
    certificateOidcIssuer: '',
    certificateIdentity: '',
    certificateTransparencyLog: {
        enabled: true,
        publicKeyPemEnc: '',
    },
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
            toggleText={toggleText}
            onToggle={onToggle}
            isExpanded={isExpanded}
            isIndented
        >
            {children}
        </ExpandableSection>
    );
};

const TransparencyLogExpandableSection = ({ toggleText, children }): ReactElement => {
    const [isExpanded, setIsExpanded] = useState(true);

    function onToggle() {
        setIsExpanded(!isExpanded);
    }

    return (
        <ExpandableSection
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
    const formInitialValues: SignatureIntegration = merge({}, defaultValues, initialValues);
    if (initialValues) {
        // To guarantee backwards compatibility with signature integrations created with ACS < 4.8,
        // we must ensure that null fields are converted to their appropriate zero values.
        const backwardsCompatibleCtlogValues: CertificateTransparencyLogVerification = {
            enabled: false,
            publicKeyPemEnc: '',
        };
        formInitialValues.cosignCertificates.forEach((item, index) => {
            formInitialValues.cosignCertificates[index].certificateTransparencyLog =
                item.certificateTransparencyLog ?? structuredClone(backwardsCompatibleCtlogValues);
        });

        const backwardsCompatibleTlogValues: TransparencyLogVerification = {
            enabled: false,
            publicKeyPemEnc: '',
            url: 'https://rekor.sigstore.dev',
            validateOffline: false,
        };
        formInitialValues.transparencyLog =
            formInitialValues.transparencyLog ?? backwardsCompatibleTlogValues;
    }
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
    const { version } = useMetadata();

    const re2syntax = (
        <>
            Supports regular expressions for matching. For more information, see{' '}
            <ExternalLink>
                <a href="https://golang.org/s/re2syntax" target="_blank" rel="noopener noreferrer">
                    RE2 syntax reference
                </a>
            </ExternalLink>
        </>
    );

    function onChange(value, event) {
        setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                <Alert
                    title="Verifying image signatures"
                    component="p"
                    variant="info"
                    isInline
                    className="pf-v5-u-mb-lg"
                >
                    <Flex direction={{ default: 'column' }}>
                        <FlexItem>
                            <Text>
                                Image signatures are verified by ensuring that their signature has
                                been signed by a trusted image signer. Configure at least one
                                trusted image signer by specifying a Cosign public encryption key or
                                a Cosign certificate chain. Multiple image signers may be combined
                                in a single signature integration.
                            </Text>
                        </FlexItem>
                        <FlexItem>
                            <Text>
                                For more information, see{' '}
                                <ExternalLink>
                                    <a
                                        href={getVersionedDocs(
                                            version,
                                            'operating/verify-image-signatures'
                                        )}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                    >
                                        RHACS documentation
                                    </a>
                                </ExternalLink>
                            </Text>
                        </FlexItem>
                    </Flex>
                </Alert>
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
                                                                grow={{ default: 'grow' }}
                                                                spaceItems={{
                                                                    default: 'spaceItemsXl',
                                                                }}
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
                                                                            style={{
                                                                                minHeight: '100px',
                                                                            }}
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
                                                                            placeholder={
                                                                                '-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----'
                                                                            }
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
                                                                grow={{ default: 'grow' }}
                                                                spaceItems={{
                                                                    default: 'spaceItemsXl',
                                                                }}
                                                            >
                                                                <FlexItem>
                                                                    <FormLabelGroup
                                                                        isRequired
                                                                        label="Certificate OIDC issuer"
                                                                        labelIcon={
                                                                            <IntegrationHelpIcon
                                                                                helpTitle="Certificate OIDC issuer"
                                                                                helpText={
                                                                                    <>
                                                                                        The
                                                                                        certificate
                                                                                        OIDC issuer
                                                                                        as specified
                                                                                        by cosign.{' '}
                                                                                        {re2syntax}{' '}
                                                                                    </>
                                                                                }
                                                                                ariaLabel="Help for certificate issuer"
                                                                            />
                                                                        }
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
                                                                        labelIcon={
                                                                            <IntegrationHelpIcon
                                                                                helpTitle="Certificate identity"
                                                                                helpText={
                                                                                    <>
                                                                                        The
                                                                                        certificate
                                                                                        identity as
                                                                                        specified by
                                                                                        cosign.{' '}
                                                                                        {re2syntax}{' '}
                                                                                    </>
                                                                                }
                                                                                ariaLabel="Help for certificate identity"
                                                                            />
                                                                        }
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
                                                                        label="Certificate chain (PEM encoded)"
                                                                        labelIcon={
                                                                            <IntegrationHelpIcon
                                                                                helpTitle="Certificate chain (PEM encoded)"
                                                                                helpText={
                                                                                    <>
                                                                                        <Text>
                                                                                            The
                                                                                            trusted
                                                                                            certificate
                                                                                            root to
                                                                                            verify
                                                                                            certificates
                                                                                            against.
                                                                                            If left
                                                                                            empty,
                                                                                            the
                                                                                            public
                                                                                            Fulcio
                                                                                            roots
                                                                                            are
                                                                                            used.
                                                                                        </Text>
                                                                                    </>
                                                                                }
                                                                                ariaLabel="Help for certificate chain PEM encoded"
                                                                            />
                                                                        }
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
                                                                            style={{
                                                                                minHeight: '100px',
                                                                            }}
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
                                                                            placeholder={
                                                                                '-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----'
                                                                            }
                                                                        />
                                                                    </FormLabelGroup>
                                                                    <FormLabelGroup
                                                                        label="Intermediate certificate (PEM encoded)"
                                                                        labelIcon={
                                                                            <IntegrationHelpIcon
                                                                                helpTitle="Intermediate certificate (PEM encoded)"
                                                                                helpText={
                                                                                    <>
                                                                                        <Text>
                                                                                            The
                                                                                            trusted
                                                                                            signer
                                                                                            intermediate
                                                                                            certificate
                                                                                            authority
                                                                                            to
                                                                                            verify
                                                                                            certificates
                                                                                            against.
                                                                                            If left
                                                                                            empty,
                                                                                            just the
                                                                                            certificate
                                                                                            chain is
                                                                                            used for
                                                                                            verification.
                                                                                        </Text>
                                                                                    </>
                                                                                }
                                                                                ariaLabel="Help for certificate chain PEM encoded"
                                                                            />
                                                                        }
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
                                                                            style={{
                                                                                minHeight: '100px',
                                                                            }}
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
                                                                            placeholder={
                                                                                '-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----'
                                                                            }
                                                                        />
                                                                    </FormLabelGroup>
                                                                </FlexItem>
                                                                <FlexItem>
                                                                    <FormLabelGroup
                                                                        fieldId={`cosignCertificates[${index}].certificateTransparencyLog.enabled`}
                                                                        helperText={
                                                                            <>
                                                                                <Text>
                                                                                    Validate the
                                                                                    proof of
                                                                                    inclusion into
                                                                                    the certificate
                                                                                    transparency
                                                                                    log.
                                                                                </Text>
                                                                            </>
                                                                        }
                                                                        touched={touched}
                                                                        errors={errors}
                                                                    >
                                                                        <Checkbox
                                                                            label="Enable certificate transparency log validation"
                                                                            id={`cosignCertificates[${index}].certificateTransparencyLog.enabled`}
                                                                            isChecked={
                                                                                values
                                                                                    .cosignCertificates[
                                                                                    index
                                                                                ]
                                                                                    ?.certificateTransparencyLog
                                                                                    ?.enabled
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
                                                                <FlexItem>
                                                                    <FormLabelGroup
                                                                        label="Certificate transparency log public key"
                                                                        fieldId={`cosignCertificates[${index}].certificateTransparencyLog.publicKeyPemEnc`}
                                                                        helperText={
                                                                            <>
                                                                                <Text>
                                                                                    The public key
                                                                                    that is used to
                                                                                    validate the
                                                                                    proof of
                                                                                    inclusion into
                                                                                    the certificate
                                                                                    transparency
                                                                                    log.
                                                                                </Text>
                                                                                <Text>
                                                                                    Leave empty to
                                                                                    use the key of
                                                                                    the public
                                                                                    Sigstore
                                                                                    instance.
                                                                                </Text>
                                                                            </>
                                                                        }
                                                                        touched={touched}
                                                                        errors={errors}
                                                                    >
                                                                        <TextArea
                                                                            autoResize
                                                                            resizeOrientation="vertical"
                                                                            type="text"
                                                                            id={`cosignCertificates[${index}].certificateTransparencyLog.publicKeyPemEnc`}
                                                                            value={
                                                                                values
                                                                                    .cosignCertificates[
                                                                                    index
                                                                                ]
                                                                                    ?.certificateTransparencyLog
                                                                                    ?.publicKeyPemEnc
                                                                            }
                                                                            style={{
                                                                                minHeight: '100px',
                                                                            }}
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
                                                                            isDisabled={
                                                                                !isEditable ||
                                                                                !values
                                                                                    .cosignCertificates[
                                                                                    index
                                                                                ]
                                                                                    ?.certificateTransparencyLog
                                                                                    ?.enabled
                                                                            }
                                                                            placeholder={
                                                                                '-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----'
                                                                            }
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
                        <TransparencyLogExpandableSection toggleText="Transparency log">
                            <Flex
                                direction={{ default: 'column' }}
                                grow={{ default: 'grow' }}
                                spaceItems={{ default: 'spaceItemsXl' }}
                            >
                                <FlexItem>
                                    <FormLabelGroup
                                        fieldId="transparencyLog.enabled"
                                        helperText={
                                            <>
                                                <Text>
                                                    Validate the inclusion of the signature in a
                                                    transparency log.
                                                </Text>
                                                <Text>
                                                    Required when signatures contain short-lived
                                                    certificates as they are issued by Fulcio.
                                                </Text>
                                            </>
                                        }
                                        touched={touched}
                                        errors={errors}
                                    >
                                        <Checkbox
                                            label="Enable transparency log validation"
                                            id="transparencyLog.enabled"
                                            isChecked={values?.transparencyLog?.enabled}
                                            onChange={(event, value) => onChange(value, event)}
                                            onBlur={handleBlur}
                                            isDisabled={!isEditable}
                                        />
                                    </FormLabelGroup>
                                </FlexItem>
                                <FlexItem>
                                    <FormLabelGroup
                                        label="Rekor URL"
                                        fieldId="transparencyLog.url"
                                        helperText={
                                            <>
                                                <Text>
                                                    The URL under which the Rekor transparency log
                                                    is available. Defaults to the public Rekor
                                                    instance of Sigstore.
                                                </Text>
                                                <Text>
                                                    Required for online confirmation of the
                                                    inclusion into the transparency log.
                                                </Text>
                                            </>
                                        }
                                        touched={touched}
                                        errors={errors}
                                    >
                                        <TextInput
                                            isRequired
                                            type="text"
                                            id="transparencyLog.url"
                                            value={values?.transparencyLog?.url}
                                            onChange={(event, value) => onChange(value, event)}
                                            onBlur={handleBlur}
                                            isDisabled={
                                                !isEditable ||
                                                !values?.transparencyLog?.enabled ||
                                                values?.transparencyLog?.validateOffline
                                            }
                                        />
                                    </FormLabelGroup>
                                </FlexItem>
                                <FlexItem>
                                    <FormLabelGroup
                                        fieldId="transparencyLog.validateOffline"
                                        touched={touched}
                                        helperText={
                                            <>
                                                <Text>
                                                    Force offline validation of the signature proof
                                                    of inclusion into the transparency log. Do not
                                                    fall back to request confirmation from the
                                                    transparency log over network.
                                                </Text>
                                            </>
                                        }
                                        errors={errors}
                                    >
                                        <Checkbox
                                            label="Validate in offline mode"
                                            id="transparencyLog.validateOffline"
                                            isChecked={values?.transparencyLog?.validateOffline}
                                            onChange={(event, value) => onChange(value, event)}
                                            onBlur={handleBlur}
                                            isDisabled={
                                                !isEditable || !values?.transparencyLog?.enabled
                                            }
                                        />
                                    </FormLabelGroup>
                                </FlexItem>
                                <FlexItem>
                                    <FormLabelGroup
                                        label="Rekor public key"
                                        fieldId={'transparencyLog.publicKeyPemEnc'}
                                        helperText={
                                            <>
                                                <Text>
                                                    The public key that is used to validate the
                                                    signature proof of inclusion into the Rekor
                                                    transparency log.
                                                </Text>
                                                <Text>
                                                    Leave empty to use the key of the public
                                                    Sigstore instance.
                                                </Text>
                                            </>
                                        }
                                        touched={touched}
                                        errors={errors}
                                    >
                                        <TextArea
                                            autoResize
                                            resizeOrientation="vertical"
                                            type="text"
                                            id={'transparencyLog.publicKeyPemEnc'}
                                            value={values?.transparencyLog?.publicKeyPemEnc}
                                            style={{ minHeight: '100px' }}
                                            onChange={(event, value) => onChange(value, event)}
                                            onBlur={handleBlur}
                                            isDisabled={
                                                !isEditable || !values?.transparencyLog?.enabled
                                            }
                                            placeholder={
                                                '-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----'
                                            }
                                        />
                                    </FormLabelGroup>
                                </FlexItem>
                            </Flex>
                        </TransparencyLogExpandableSection>
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
