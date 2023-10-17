import React from 'react';
import {
    Bullseye,
    Button,
    DatePicker,
    Flex,
    FormGroup,
    Form,
    Radio,
    Spinner,
    Tabs,
    Tab,
    TextArea,
    Text,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { useFormik } from 'formik';
import { addDays } from 'date-fns';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { fetchVulnerabilitiesExceptionConfig } from 'services/ExceptionConfigService';
import useRestQuery from 'hooks/useRestQuery';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import {
    DeferralValues,
    ScopeContext,
    deferralValidationSchema,
    futureDateValidator,
} from './utils';
import ExceptionScopeField from './ExceptionScopeField';
import CveSelections from './CveSelections';

function getDefaultValues(cves: string[], scopeContext: ScopeContext): DeferralValues {
    const imageScope =
        scopeContext === 'GLOBAL'
            ? { registry: '.*', remote: '.*', tag: '.*' }
            : { registry: '.*', remote: scopeContext.image.name, tag: '.*' };

    return {
        cves,
        comment: '',
        scope: {
            imageScope,
        },
    };
}

/**
 * Returns the date portion of an ISO date string
 * @param date - ISO date string
 * @returns Date portion of the ISO date string, or an empty string if the date is falsy
 */
function prettifyDate(date = ''): string {
    return date.substring(0, 10);
}

export type DeferralFormProps = {
    cves: string[];
    scopeContext: ScopeContext;
    onCancel: () => void;
};

function DeferralForm({ cves, scopeContext, onCancel }: DeferralFormProps) {
    const { data: config, loading, error } = useRestQuery(fetchVulnerabilitiesExceptionConfig);

    const formik = useFormik({
        initialValues: getDefaultValues(cves, scopeContext),
        onSubmit: () => {},
        validationSchema: deferralValidationSchema,
    });
    const { values, setValues, setFieldValue, handleBlur, touched, errors } = formik;

    if (loading) {
        return (
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        );
    }

    if (error) {
        return (
            <Bullseye>
                <EmptyStateTemplate
                    headingLevel="h2"
                    title="There was an error loading the vulnerability exception configuration"
                    icon={ExclamationCircleIcon}
                    iconClassName="pf-u-danger-color-100"
                >
                    {getAxiosErrorMessage(error)}
                </EmptyStateTemplate>
            </Bullseye>
        );
    }

    function setExpiry(expiry: DeferralValues['expiry']) {
        setValues((prev) => ({ ...prev, expiry })).catch(() => {});
    }

    return (
        <>
            <Form>
                <Tabs defaultActiveKey="options">
                    <Tab eventKey="options" title="Options">
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsLg' }}
                        >
                            <Text>CVEs will be marked as deferred after approval</Text>
                            {config && (
                                <FormGroup label="How long should the CVEs be deferred?" isRequired>
                                    <Flex
                                        direction={{ default: 'column' }}
                                        spaceItems={{ default: 'spaceItemsXs' }}
                                    >
                                        {config.expiryOptions.fixableCveOptions.anyFixable && (
                                            <Radio
                                                id="any-cve-fixable"
                                                name="any-cve-fixable"
                                                isChecked={
                                                    values.expiry?.type === 'ANY_CVE_FIXABLE'
                                                }
                                                onChange={() => {
                                                    setExpiry({ type: 'ANY_CVE_FIXABLE' });
                                                }}
                                                label="When any CVE is fixable"
                                            />
                                        )}
                                        {config.expiryOptions.fixableCveOptions.allFixable && (
                                            <Radio
                                                id="all-cve-fixable"
                                                name="all-cve-fixable"
                                                isChecked={
                                                    values.expiry?.type === 'ALL_CVE_FIXABLE'
                                                }
                                                onChange={() => {
                                                    setExpiry({ type: 'ALL_CVE_FIXABLE' });
                                                }}
                                                label="When all CVEs are fixable"
                                            />
                                        )}
                                        {config.expiryOptions.dayOptions
                                            .filter((option) => option.enabled)
                                            .map(({ numDays }) => (
                                                <Radio
                                                    id={`fixed-duration-${numDays}`}
                                                    name={`fixed-duration-${numDays}`}
                                                    key={`fixed-duration-${numDays}`}
                                                    isChecked={
                                                        values.expiry?.type === 'TIME' &&
                                                        values.expiry?.days === numDays
                                                    }
                                                    onChange={() => {
                                                        setExpiry({ type: 'TIME', days: numDays });
                                                    }}
                                                    label={`For ${numDays} days`}
                                                />
                                            ))}
                                        {/* TODO - Awaiting backend support for indefinite deferrals
                                         config.expiryOptions.indefinite && (
                                            <Radio
                                                id="indefinite"
                                                name="indefinite"
                                                isChecked={values.expiryType === 'INDEFINITE'}
                                                onChange={() => {}}
                                                label="Indefinitely"
                                            />
                                        )
                                        */}
                                        {config.expiryOptions.customDate && (
                                            <Radio
                                                id="custom-date"
                                                name="custom-date"
                                                isChecked={values.expiry?.type === 'CUSTOM_DATE'}
                                                onChange={() =>
                                                    setExpiry({
                                                        type: 'CUSTOM_DATE',
                                                        date: addDays(new Date(), 1).toISOString(),
                                                    })
                                                }
                                                label="Until a specific date"
                                            />
                                        )}

                                        {config.expiryOptions.customDate &&
                                            values.expiry?.type === 'CUSTOM_DATE' && (
                                                <div>
                                                    <DatePicker
                                                        name="custom-date-picker"
                                                        value={prettifyDate(values.expiry?.date)}
                                                        onChange={(_, value) => {
                                                            setExpiry({
                                                                type: 'CUSTOM_DATE',
                                                                date: value,
                                                            });
                                                        }}
                                                        validators={[futureDateValidator]}
                                                    />
                                                </div>
                                            )}
                                    </Flex>
                                </FormGroup>
                            )}
                            <ExceptionScopeField
                                fieldId="scope"
                                label="Scope"
                                formik={formik}
                                scopeContext={scopeContext}
                            />
                            <FormGroup fieldId="comment" label="Deferral rationale" isRequired>
                                <TextArea
                                    id="comment"
                                    name="comment"
                                    isRequired
                                    onBlur={handleBlur('comment')}
                                    onChange={(value) => setFieldValue('comment', value)}
                                    validated={
                                        touched.comment && errors.comment ? 'error' : 'default'
                                    }
                                />
                            </FormGroup>
                        </Flex>
                    </Tab>
                    <Tab eventKey="cves" title="CVE Selections">
                        <CveSelections cves={cves} />
                    </Tab>
                </Tabs>
                <Flex>
                    <Button onClick={() => {}}>Submit request</Button>
                    <Button variant="secondary" onClick={onCancel}>
                        Cancel
                    </Button>
                </Flex>
            </Form>
        </>
    );
}

export default DeferralForm;
