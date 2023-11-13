import React from 'react';
import {
    Bullseye,
    DatePicker,
    FormGroup,
    FormHelperText,
    Flex,
    HelperText,
    HelperTextItem,
    Radio,
    Spinner,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { addDays } from 'date-fns';
import { useFormik } from 'formik';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import useRestQuery from 'hooks/useRestQuery';
import { fetchVulnerabilitiesExceptionConfig } from 'services/ExceptionConfigService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { DeferralValues, futureDateValidator } from './utils';

/**
 * Returns the date portion of an ISO date string
 * @param date - ISO date string
 * @returns Date portion of the ISO date string, or an empty string if the date is falsy
 */
function prettifyDate(date = ''): string {
    return date.substring(0, 10);
}

export type ExpiryFieldProps = {
    formik: ReturnType<typeof useFormik<DeferralValues>>;
};

function ExpiryField({ formik }: ExpiryFieldProps) {
    const { data: config, loading, error } = useRestQuery(fetchVulnerabilitiesExceptionConfig);
    const { values, errors, setValues } = formik;

    function setExpiry(expiry: DeferralValues['expiry']) {
        return setValues((prev) => ({ ...prev, expiry }));
    }

    if (loading || !config) {
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

    return (
        <FormGroup label="How long should the CVEs be deferred?" isRequired>
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
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
                                values.expiry?.type === 'TIME' && values.expiry?.days === numDays
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
                {config.expiryOptions.indefinite && (
                    <Radio
                        id="indefinite"
                        name="indefinite"
                        isChecked={values.expiry?.type === 'INDEFINITE'}
                        onChange={() => setExpiry({ type: 'INDEFINITE' })}
                        label="Indefinitely"
                    />
                )}
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

                {config.expiryOptions.customDate && values.expiry?.type === 'CUSTOM_DATE' && (
                    <div>
                        <DatePicker
                            name="custom-date-picker"
                            value={prettifyDate(values.expiry.date)}
                            onChange={(_, value) =>
                                setExpiry({
                                    type: 'CUSTOM_DATE',
                                    date: new Date(value).toISOString(),
                                })
                            }
                            validators={[futureDateValidator]}
                        />
                    </div>
                )}
                {errors.expiry && (
                    <FormHelperText isError isHidden={false}>
                        <HelperText>
                            <HelperTextItem
                                variant="error"
                                icon={<ExclamationCircleIcon />}
                                className="pf-u-display-flex pf-u-align-items-center"
                            >
                                {errors.expiry}
                            </HelperTextItem>
                        </HelperText>
                    </FormHelperText>
                )}
            </Flex>
        </FormGroup>
    );
}

export default ExpiryField;
