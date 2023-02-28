import React, { useState } from 'react';
import { Button, Badge, Modal, Form, FormGroup, Checkbox, Flex } from '@patternfly/react-core';
import cloneDeep from 'lodash/cloneDeep';
import { useFormik, FormikProvider } from 'formik';
import { Globe } from 'react-feather';

import { VulnerabilitySeverity } from 'types/cve.proto';
import { DefaultFilters, FixableStatus } from './types';

type DefaultFilterModalProps = {
    defaultFilters: DefaultFilters;
    setLocalStorage: (values) => void;
};

function DefaultFilterModal({ defaultFilters, setLocalStorage }: DefaultFilterModalProps) {
    const [isOpen, setIsOpen] = useState(false);
    const totalFilters = defaultFilters.Severity.length + defaultFilters.Fixable.length;

    const formik = useFormik({
        initialValues: cloneDeep(defaultFilters),
        onSubmit: (values: DefaultFilters) => {
            setLocalStorage(values);
            setIsOpen(false);
        },
    });

    const { submitForm, values, setFieldValue, setValues } = formik;
    const severityValues = values.Severity;
    const fixableValues = values.Fixable;

    function handleModalToggle() {
        if (isOpen) {
            setValues(defaultFilters).catch(() => {});
        }
        setIsOpen(!isOpen);
    }

    function handleSeverityChange(severity: VulnerabilitySeverity, isChecked: boolean) {
        let newSeverityValues = [...severityValues];
        if (isChecked) {
            newSeverityValues.push(severity);
        } else {
            newSeverityValues = newSeverityValues.filter((val) => val !== severity);
        }
        setFieldValue('Severity', newSeverityValues).catch(() => {});
    }

    function handleFixableChange(fixable: FixableStatus, isChecked: boolean) {
        let newFixableValues = [...fixableValues];
        if (isChecked) {
            newFixableValues.push(fixable);
        } else {
            newFixableValues = newFixableValues.filter((val) => val !== fixable);
        }
        setFieldValue('Fixable', newFixableValues).catch(() => {});
    }

    return (
        <>
            <Button variant="plain" onClick={handleModalToggle}>
                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <Globe className="pf-u-mr-sm" />
                    Default vulnerability filters
                    <Badge key={1} isRead className="pf-u-ml-sm">
                        {totalFilters}
                    </Badge>
                </Flex>
            </Button>
            <Modal
                title="Default vulnerability filters"
                description="Select default vulnerability filters to be applied across all views."
                isOpen={isOpen}
                onClose={handleModalToggle}
                variant="medium"
                actions={[
                    <Button key="apply" variant="primary" onClick={submitForm}>
                        Apply filters
                    </Button>,
                    <Button key="cancel" variant="link" onClick={handleModalToggle}>
                        Cancel
                    </Button>,
                ]}
            >
                <FormikProvider value={formik}>
                    <Form id="default-filter-modal-form">
                        <FormGroup label="CVE severity" isInline>
                            <Checkbox
                                label="Critical"
                                id="critical-severity"
                                isChecked={severityValues.includes(
                                    'CRITICAL_VULNERABILITY_SEVERITY'
                                )}
                                onChange={(isChecked) => {
                                    handleSeverityChange(
                                        'CRITICAL_VULNERABILITY_SEVERITY',
                                        isChecked
                                    );
                                }}
                            />
                            <Checkbox
                                label="Important"
                                id="important-severity"
                                isChecked={severityValues.includes(
                                    'IMPORTANT_VULNERABILITY_SEVERITY'
                                )}
                                onChange={(isChecked) => {
                                    handleSeverityChange(
                                        'IMPORTANT_VULNERABILITY_SEVERITY',
                                        isChecked
                                    );
                                }}
                            />
                            <Checkbox
                                label="Moderate"
                                id="moderate-severity"
                                isChecked={severityValues.includes(
                                    'MODERATE_VULNERABILITY_SEVERITY'
                                )}
                                onChange={(isChecked) => {
                                    handleSeverityChange(
                                        'MODERATE_VULNERABILITY_SEVERITY',
                                        isChecked
                                    );
                                }}
                            />
                            <Checkbox
                                label="Low"
                                id="low-severity"
                                isChecked={severityValues.includes('LOW_VULNERABILITY_SEVERITY')}
                                onChange={(isChecked) => {
                                    handleSeverityChange('LOW_VULNERABILITY_SEVERITY', isChecked);
                                }}
                            />
                        </FormGroup>
                        <FormGroup label="CVE status" isInline>
                            <Checkbox
                                label="Fixable"
                                id="fixable-status"
                                isChecked={fixableValues.includes('Fixable')}
                                onChange={(isChecked) => {
                                    handleFixableChange('Fixable', isChecked);
                                }}
                            />
                            <Checkbox
                                label="Not fixable"
                                id="not-fixable-status"
                                isChecked={fixableValues.includes('Not fixable')}
                                onChange={(isChecked) => {
                                    handleFixableChange('Not fixable', isChecked);
                                }}
                            />
                        </FormGroup>
                    </Form>
                </FormikProvider>
            </Modal>
        </>
    );
}

export default DefaultFilterModal;
