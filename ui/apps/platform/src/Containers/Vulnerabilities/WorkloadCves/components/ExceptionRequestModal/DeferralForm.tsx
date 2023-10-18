import React, { useState } from 'react';
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
    TabContent,
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
import ExceptionScopeField, { ALL } from './ExceptionScopeField';
import CveSelections, { CveSelectionsProps } from './CveSelections';

function getDefaultValues(cves: string[], scopeContext: ScopeContext): DeferralValues {
    const imageScope =
        scopeContext === 'GLOBAL'
            ? { registry: ALL, remote: ALL, tag: ALL }
            : { registry: ALL, remote: scopeContext.image.name, tag: ALL };

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
    cves: CveSelectionsProps['cves'];
    scopeContext: ScopeContext;
    onCancel: () => void;
};

function DeferralForm({ cves, scopeContext, onCancel }: DeferralFormProps) {
    const [activeKeyTab, setActiveKeyTab] = useState<string | number>('options');
    const { data: config, loading, error } = useRestQuery(fetchVulnerabilitiesExceptionConfig);

    const formik = useFormik({
        initialValues: getDefaultValues(
            cves.map(({ cve }) => cve),
            scopeContext
        ),
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
        return setValues((prev) => ({ ...prev, expiry }));
    }

    return (
        <>
            <Form className="pf-u-display-flex pf-u-flex-direction-column" style={{ minHeight: 0 }}>
                <Tabs
                    className="pf-u-flex-shrink-0"
                    activeKey={activeKeyTab}
                    onSelect={(_, tab) => setActiveKeyTab(tab)}
                >
                    <Tab eventKey="options" title="Options" tabContentId="options" />
                    <Tab eventKey="cves" title="CVE selections" tabContentId="cves" />
                </Tabs>
                <TabContent
                    id="options"
                    className="pf-u-flex-1"
                    hidden={activeKeyTab !== 'options'}
                >
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
                                            isChecked={values.expiry?.type === 'ANY_CVE_FIXABLE'}
                                            onChange={() => setExpiry({ type: 'ANY_CVE_FIXABLE' })}
                                            label="When any CVE is fixable"
                                        />
                                    )}
                                    {config.expiryOptions.fixableCveOptions.allFixable && (
                                        <Radio
                                            id="all-cve-fixable"
                                            name="all-cve-fixable"
                                            isChecked={values.expiry?.type === 'ALL_CVE_FIXABLE'}
                                            onChange={() => setExpiry({ type: 'ALL_CVE_FIXABLE' })}
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
                                                onChange={() =>
                                                    setExpiry({
                                                        type: 'TIME',
                                                        days: numDays,
                                                    })
                                                }
                                                label={`For ${numDays} days`}
                                            />
                                        ))}
                                    {/* TODO - Awaiting backend support for indefinite deferrals
                                         config.expiryOptions.indefinite && (
                                            <Radio
                                                id="indefinite"
                                                name="indefinite"
                                                isChecked={values.expiry?.type === 'INDEFINITE'}
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
                                                    value={prettifyDate(values.expiry.date)}
                                                    onChange={(_, value) =>
                                                        setExpiry({
                                                            type: 'CUSTOM_DATE',
                                                            date: value,
                                                        })
                                                    }
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
                                validated={touched.comment && errors.comment ? 'error' : 'default'}
                            />
                        </FormGroup>
                    </Flex>
                </TabContent>
                <TabContent
                    id="cves"
                    className="pf-u-flex-1"
                    hidden={activeKeyTab !== 'cves'}
                    style={{ overflowY: 'auto' }}
                >
                    <CveSelections cves={cves} />
                </TabContent>
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
