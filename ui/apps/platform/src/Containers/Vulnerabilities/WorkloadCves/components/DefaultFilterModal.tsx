import React, { useState } from 'react';
import { Button, Modal, Form, FormGroup, Checkbox } from '@patternfly/react-core';
import cloneDeep from 'lodash/cloneDeep';
import { useFormik, FormikProvider } from 'formik';
import { Globe } from 'react-feather';

import useAnalytics, { WORKLOAD_CVE_DEFAULT_FILTERS_CHANGED } from 'hooks/useAnalytics';
import { DefaultFilters, FixableStatus, VulnerabilitySeverityLabel } from '../../types';

function analyticsTrackDefaultFilters(
    analyticsTrack: ReturnType<typeof useAnalytics>['analyticsTrack'],
    filters: DefaultFilters
) {
    analyticsTrack({
        event: WORKLOAD_CVE_DEFAULT_FILTERS_CHANGED,
        properties: {
            SEVERITY_CRITICAL: filters.SEVERITY.includes('Critical') ? 1 : 0,
            SEVERITY_IMPORTANT: filters.SEVERITY.includes('Important') ? 1 : 0,
            SEVERITY_MODERATE: filters.SEVERITY.includes('Moderate') ? 1 : 0,
            SEVERITY_LOW: filters.SEVERITY.includes('Low') ? 1 : 0,
            SEVERITY_UNKNOWN: filters.SEVERITY.includes('Unknown') ? 1 : 0,
            CVE_STATUS_FIXABLE: filters.FIXABLE.includes('Fixable') ? 1 : 0,
            CVE_STATUS_NOT_FIXABLE: filters.FIXABLE.includes('Not fixable') ? 1 : 0,
        },
    });
}

type DefaultFilterModalProps = {
    defaultFilters: DefaultFilters;
    setLocalStorage: (values: DefaultFilters) => void;
};

function DefaultFilterModal({ defaultFilters, setLocalStorage }: DefaultFilterModalProps) {
    const { analyticsTrack } = useAnalytics();
    const [isOpen, setIsOpen] = useState(false);
    const totalFilters = defaultFilters.SEVERITY.length + defaultFilters.FIXABLE.length;

    const formik = useFormik({
        initialValues: cloneDeep(defaultFilters),
        onSubmit: (values: DefaultFilters) => {
            setLocalStorage(values);
            setIsOpen(false);
            analyticsTrackDefaultFilters(analyticsTrack, values);
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
            <Button
                variant="secondary"
                className="pf-v5-u-display-inline-flex pf-v5-u-align-items-center"
                onClick={handleModalToggle}
                countOptions={{
                    isRead: true,
                    count: totalFilters,
                    className: 'custom-badge-unread',
                }}
            >
                <Globe height="20px" width="20px" className="pf-v5-u-mr-sm" />
                <span>Default filters</span>
            </Button>
            <Modal
                title="Default filters"
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
                                onChange={(_event, isChecked) => {
                                    handleSeverityChange('Critical', isChecked);
                                }}
                            />
                            <Checkbox
                                label="Important"
                                id="important-severity"
                                isChecked={severityValues.includes('Important')}
                                onChange={(_event, isChecked) => {
                                    handleSeverityChange('Important', isChecked);
                                }}
                            />
                            <Checkbox
                                label="Moderate"
                                id="moderate-severity"
                                isChecked={severityValues.includes('Moderate')}
                                onChange={(_event, isChecked) => {
                                    handleSeverityChange('Moderate', isChecked);
                                }}
                            />
                            <Checkbox
                                label="Low"
                                id="low-severity"
                                isChecked={severityValues.includes('Low')}
                                onChange={(_event, isChecked) => {
                                    handleSeverityChange('Low', isChecked);
                                }}
                            />
                            <Checkbox
                                label="Unknown"
                                id="unknown-severity"
                                isChecked={severityValues.includes('Unknown')}
                                onChange={(_event, isChecked) => {
                                    handleSeverityChange('Unknown', isChecked);
                                }}
                            />
                        </FormGroup>
                        <FormGroup label="CVE status" isInline>
                            <Checkbox
                                label="Fixable"
                                id="fixable-status"
                                isChecked={fixableValues.includes('Fixable')}
                                onChange={(_event, isChecked) => {
                                    handleFixableChange('Fixable', isChecked);
                                }}
                            />
                            <Checkbox
                                label="Not fixable"
                                id="not-fixable-status"
                                isChecked={fixableValues.includes('Not fixable')}
                                onChange={(_event, isChecked) => {
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
