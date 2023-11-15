import React, { useState } from 'react';
import { Button, Badge, Modal, Form, FormGroup, Checkbox, Flex } from '@patternfly/react-core';
import cloneDeep from 'lodash/cloneDeep';
import { useFormik, FormikProvider } from 'formik';
import { Globe } from 'react-feather';

import { DefaultFilters, FixableStatus, VulnerabilitySeverityLabel } from '../types';

type DefaultFilterModalProps = {
    defaultFilters: DefaultFilters;
    setLocalStorage: (values: DefaultFilters) => void;
};

function DefaultFilterModal({ defaultFilters, setLocalStorage }: DefaultFilterModalProps) {
    const [isOpen, setIsOpen] = useState(false);
    const totalFilters = defaultFilters.SEVERITY.length + defaultFilters.FIXABLE.length;

    const formik = useFormik({
        initialValues: cloneDeep(defaultFilters),
        onSubmit: (values: DefaultFilters) => {
            setLocalStorage(values);
            setIsOpen(false);
        },
    });

    const { submitForm, values, setFieldValue, setValues } = formik;
    const severityValues = values.SEVERITY;
    const fixableValues = values.FIXABLE;

    function handleModalToggle() {
        if (isOpen) {
            setValues(defaultFilters).catch(() => {});
        }
        setIsOpen(!isOpen);
    }

    function handleSeverityChange(severity: VulnerabilitySeverityLabel, isChecked: boolean) {
        let newSeverityValues = [...severityValues];
        if (isChecked) {
            newSeverityValues.push(severity);
        } else {
            newSeverityValues = newSeverityValues.filter((val) => val !== severity);
        }
        setFieldValue('SEVERITY', newSeverityValues).catch(() => {});
    }

    function handleFixableChange(fixable: FixableStatus, isChecked: boolean) {
        let newFixableValues = [...fixableValues];
        if (isChecked) {
            newFixableValues.push(fixable);
        } else {
            newFixableValues = newFixableValues.filter((val) => val !== fixable);
        }
        setFieldValue('FIXABLE', newFixableValues).catch(() => {});
    }

    return (
        <>
            <Button variant="plain" className="pf-u-color-300" onClick={handleModalToggle}>
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
                                isChecked={severityValues.includes('Critical')}
                                onChange={(isChecked) => {
                                    handleSeverityChange('Critical', isChecked);
                                }}
                            />
                            <Checkbox
                                label="Important"
                                id="important-severity"
                                isChecked={severityValues.includes('Important')}
                                onChange={(isChecked) => {
                                    handleSeverityChange('Important', isChecked);
                                }}
                            />
                            <Checkbox
                                label="Moderate"
                                id="moderate-severity"
                                isChecked={severityValues.includes('Moderate')}
                                onChange={(isChecked) => {
                                    handleSeverityChange('Moderate', isChecked);
                                }}
                            />
                            <Checkbox
                                label="Low"
                                id="low-severity"
                                isChecked={severityValues.includes('Low')}
                                onChange={(isChecked) => {
                                    handleSeverityChange('Low', isChecked);
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
