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

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { fetchVulnerabilitiesExceptionConfig } from 'services/ExceptionConfigService';
import useRestQuery from 'hooks/useRestQuery';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { ScopeContext } from './utils';
import ExceptionScopeField from './ExceptionScopeField';
import CveSelections from './CveSelections';

export type DeferralFormProps = {
    cves: string[];
    scopeContext: ScopeContext;
    onCancel: () => void;
};

function DeferralForm({ cves, scopeContext, onCancel }: DeferralFormProps) {
    const { data: config, loading, error } = useRestQuery(fetchVulnerabilitiesExceptionConfig);

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
                                                isChecked={false}
                                                onChange={() => {}}
                                                label="When any CVE is fixable"
                                            />
                                        )}
                                        {config.expiryOptions.fixableCveOptions.allFixable && (
                                            <Radio
                                                id="all-cve-fixable"
                                                name="all-cve-fixable"
                                                isChecked={false}
                                                onChange={() => {}}
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
                                                    isChecked={false}
                                                    onChange={() => {}}
                                                    label={`For ${numDays} days`}
                                                />
                                            ))}
                                        {/* TODO - Awaiting backend support for indefinite deferrals
                                         config.expiryOptions.indefinite && (
                                            <Radio
                                                id="indefinite"
                                                name="duration"
                                                isChecked={false}
                                                onChange={() => {}}
                                                label="Indefinitely"
                                            />
                                        )
                                        */}
                                        {config.expiryOptions.customDate && (
                                            <Radio
                                                id="custom-date"
                                                name="custom-date"
                                                isChecked={false}
                                                onChange={() => {}}
                                                label="Until a specific date"
                                            />
                                        )}
                                        {config.expiryOptions.customDate && false && (
                                            <div>
                                                <DatePicker name="custom-date-picker" />
                                            </div>
                                        )}
                                    </Flex>
                                </FormGroup>
                            )}
                            <ExceptionScopeField
                                fieldId="scope"
                                label="Scope"
                                scopeContext={scopeContext}
                            />
                            <FormGroup fieldId="comment" label="Deferral rationale" isRequired>
                                <TextArea id="comment" name="comment" isRequired />
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
