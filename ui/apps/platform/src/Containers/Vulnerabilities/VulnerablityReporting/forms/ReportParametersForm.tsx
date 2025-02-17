import React, { ChangeEvent, FormEvent, ReactElement } from 'react';
import {
    Checkbox,
    DatePicker,
    Divider,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    PageSection,
    TextArea,
    TextInput,
    Title,
} from '@patternfly/react-core';
import { SelectOption } from '@patternfly/react-core/deprecated';
import { FormikProps } from 'formik';
import { cloneDeep } from 'lodash';

import {
    CVESDiscoveredSince,
    ReportFormValues,
} from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';
import { fixabilityLabels } from 'constants/reportConstants';
import {
    cvesDiscoveredSinceLabelMap,
    imageTypeLabelMap,
} from 'Containers/Vulnerabilities/VulnerablityReporting/utils';

import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import SelectSingle from 'Components/SelectSingle/SelectSingle';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { CollectionSlim } from 'services/CollectionsService';
import { NotifierConfiguration } from 'services/ReportsService.types';
import CollectionSelection from './CollectionSelection';

export type ReportParametersFormParams = {
    title: string;
    formik: FormikProps<ReportFormValues>;
};

function ReportParametersForm({ title, formik }: ReportParametersFormParams): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isIncludeAdvisoryEnabled =
        isFeatureFlagEnabled('ROX_SCANNER_V4') &&
        isFeatureFlagEnabled('ROX_CVE_ADVISORY_SEPARATION');
    const isIncludeEpssProbabilityEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const isIncludeNvdCvssEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');

    const handleTextChange =
        (fieldName: string) =>
        (event: FormEvent<HTMLInputElement> | ChangeEvent<HTMLTextAreaElement>, value: string) => {
            formik.setFieldValue(fieldName, value);
        };

    const handleCheckboxSelectChange = (fieldName: string) => (selection: string[]) => {
        formik.setFieldValue(fieldName, selection);
    };

    const handleDateSelection =
        (fieldName: string) => (_event: React.FormEvent, selection: string) => {
            formik.setFieldValue(fieldName, selection);
        };

    const handleCollectionSelection = (fieldName: string) => (selection: CollectionSlim | null) => {
        formik.setFieldValue(fieldName, selection);
    };

    const handleCVEsDiscoveredStartDate = handleDateSelection(
        'reportParameters.cvesDiscoveredStartDate'
    );

    function onChange(event, value) {
        return formik.setFieldValue(event.target.id, value, false);
    }

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v5-u-py-lg pf-v5-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">{title}</Title>
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Form className="pf-v5-u-py-lg pf-v5-u-px-lg">
                <FormLabelGroup
                    label="Report name"
                    isRequired
                    fieldId="reportParameters.reportName"
                    errors={formik.errors}
                >
                    <TextInput
                        isRequired
                        type="text"
                        id="reportParameters.reportName"
                        name="reportParameters.reportName"
                        value={formik.values.reportParameters.reportName}
                        onChange={handleTextChange('reportParameters.reportName')}
                        onBlur={formik.handleBlur}
                    />
                </FormLabelGroup>
                <FormLabelGroup
                    label="Report description"
                    fieldId="reportParameters.reportDescription"
                    errors={formik.errors}
                >
                    <TextArea
                        type="text"
                        id="reportParameters.reportDescription"
                        name="reportParameters.reportDescription"
                        value={formik.values.reportParameters.reportDescription}
                        onChange={handleTextChange('reportParameters.reportDescription')}
                        onBlur={formik.handleBlur}
                    />
                </FormLabelGroup>
                <FormLabelGroup
                    label="CVE severity"
                    isRequired
                    fieldId="reportParameters.cveSeverities"
                    errors={formik.errors}
                >
                    <CheckboxSelect
                        toggleId="reportParameters.cveSeverities"
                        name="reportParameters.cveSeverities"
                        ariaLabel="CVE severity checkbox select"
                        selections={formik.values.reportParameters.cveSeverities}
                        onChange={handleCheckboxSelectChange('reportParameters.cveSeverities')}
                        onBlur={formik.handleBlur}
                        placeholderText="CVE severity"
                        menuAppendTo={() => document.body}
                    >
                        <SelectOption value="CRITICAL_VULNERABILITY_SEVERITY">
                            <Flex
                                className="pf-v5-u-mx-sm"
                                spaceItems={{ default: 'spaceItemsSm' }}
                                alignItems={{ default: 'alignItemsCenter' }}
                            >
                                <VulnerabilitySeverityIconText severity="CRITICAL_VULNERABILITY_SEVERITY" />
                            </Flex>
                        </SelectOption>
                        <SelectOption value="IMPORTANT_VULNERABILITY_SEVERITY">
                            <Flex
                                className="pf-v5-u-mx-sm"
                                spaceItems={{ default: 'spaceItemsSm' }}
                                alignItems={{ default: 'alignItemsCenter' }}
                            >
                                <VulnerabilitySeverityIconText severity="IMPORTANT_VULNERABILITY_SEVERITY" />
                            </Flex>
                        </SelectOption>
                        <SelectOption value="MODERATE_VULNERABILITY_SEVERITY">
                            <Flex
                                className="pf-v5-u-mx-sm"
                                spaceItems={{ default: 'spaceItemsSm' }}
                                alignItems={{ default: 'alignItemsCenter' }}
                            >
                                <VulnerabilitySeverityIconText severity="MODERATE_VULNERABILITY_SEVERITY" />
                            </Flex>
                        </SelectOption>
                        <SelectOption value="LOW_VULNERABILITY_SEVERITY">
                            <Flex
                                className="pf-v5-u-mx-sm"
                                spaceItems={{ default: 'spaceItemsSm' }}
                                alignItems={{ default: 'alignItemsCenter' }}
                            >
                                <VulnerabilitySeverityIconText severity="LOW_VULNERABILITY_SEVERITY" />
                            </Flex>
                        </SelectOption>
                    </CheckboxSelect>
                </FormLabelGroup>
                <FormLabelGroup
                    label="CVE status"
                    isRequired
                    fieldId="reportParameters.cveStatus"
                    errors={formik.errors}
                >
                    <CheckboxSelect
                        toggleId="reportParameters.cveStatus"
                        name="reportParameters.cveStatus"
                        ariaLabel="CVE status checkbox select"
                        selections={formik.values.reportParameters.cveStatus}
                        onChange={handleCheckboxSelectChange('reportParameters.cveStatus')}
                        onBlur={formik.handleBlur}
                        placeholderText="CVE status"
                        menuAppendTo={() => document.body}
                    >
                        <SelectOption value="FIXABLE">{fixabilityLabels.FIXABLE}</SelectOption>
                        <SelectOption value="NOT_FIXABLE">
                            {fixabilityLabels.NOT_FIXABLE}
                        </SelectOption>
                    </CheckboxSelect>
                </FormLabelGroup>
                <FormLabelGroup
                    label="Image type"
                    isRequired
                    fieldId="reportParameters.imageType"
                    errors={formik.errors}
                >
                    <CheckboxSelect
                        toggleId="reportParameters.imageType"
                        name="reportParameters.imageType"
                        ariaLabel="Image type checkbox select"
                        selections={formik.values.reportParameters.imageType}
                        onChange={handleCheckboxSelectChange('reportParameters.imageType')}
                        onBlur={formik.handleBlur}
                        placeholderText="Image type"
                        menuAppendTo={() => document.body}
                    >
                        <SelectOption value="DEPLOYED">{imageTypeLabelMap.DEPLOYED}</SelectOption>
                        <SelectOption value="WATCHED">{imageTypeLabelMap.WATCHED}</SelectOption>
                    </CheckboxSelect>
                </FormLabelGroup>
                <FormLabelGroup
                    label="CVEs discovered since"
                    isRequired
                    fieldId="reportParameters.cvesDiscoveredSince"
                    errors={formik.errors}
                >
                    <SelectSingle
                        id="reportParameters.cvesDiscoveredSince"
                        value={formik.values.reportParameters.cvesDiscoveredSince}
                        handleSelect={(name: string, value: string) => {
                            const newCVEsDiscoveredSinceValue = value as CVESDiscoveredSince;
                            const modifiedFormValues = cloneDeep(formik.values);

                            if (
                                modifiedFormValues.deliveryDestinations.length === 0 &&
                                newCVEsDiscoveredSinceValue === 'SINCE_LAST_REPORT'
                            ) {
                                // since delivery destinations are required in this case, we will
                                // automatically add to the array so the user doesn't need to do it
                                // manually
                                const newDeliveryDestination: NotifierConfiguration = {
                                    emailConfig: {
                                        notifierId: '',
                                        mailingLists: [],
                                        customSubject: '',
                                        customBody: '',
                                    },
                                    notifierName: '',
                                };
                                modifiedFormValues.deliveryDestinations.push(
                                    newDeliveryDestination
                                );
                            }
                            modifiedFormValues.reportParameters.cvesDiscoveredSince =
                                newCVEsDiscoveredSinceValue;

                            formik.setValues(modifiedFormValues);
                        }}
                        onBlur={formik.handleBlur}
                        menuAppendTo={() => document.body}
                    >
                        <SelectOption
                            value="SINCE_LAST_REPORT"
                            description="At least one delivery destination and schedule will be required in the next step."
                        >
                            {cvesDiscoveredSinceLabelMap.SINCE_LAST_REPORT}
                        </SelectOption>
                        <SelectOption
                            value="START_DATE"
                            description="Custom start date for the discovered CVE that were run on-demand or downloaded"
                        >
                            {cvesDiscoveredSinceLabelMap.START_DATE}
                        </SelectOption>
                        <SelectOption
                            value="ALL_VULN"
                            description="Show all detected CVEs from the beginning of cluster setup"
                        >
                            {cvesDiscoveredSinceLabelMap.ALL_VULN}
                        </SelectOption>
                    </SelectSingle>
                </FormLabelGroup>
                {formik.values.reportParameters.cvesDiscoveredSince === 'START_DATE' && (
                    <FormLabelGroup
                        isRequired
                        fieldId="reportParameters.cvesDiscoveredStartDate"
                        errors={formik.errors}
                    >
                        <DatePicker
                            name="reportParameters.cvesDiscoveredStartDate"
                            value={formik.values.reportParameters.cvesDiscoveredStartDate}
                            onBlur={formik.handleBlur}
                            onChange={handleCVEsDiscoveredStartDate}
                        />
                    </FormLabelGroup>
                )}
                {(isIncludeNvdCvssEnabled ||
                    isIncludeEpssProbabilityEnabled ||
                    isIncludeAdvisoryEnabled) && (
                    <FormGroup label="Optional columns" isInline isStack>
                        {isIncludeNvdCvssEnabled && (
                            <Checkbox
                                label="Include NVD CVSS"
                                id="reportParameters.includeNvdCvss"
                                isChecked={formik.values.reportParameters.includeNvdCvss}
                                onChange={onChange}
                            />
                        )}
                        {isIncludeEpssProbabilityEnabled && (
                            <Checkbox
                                label="Include EPSS probability"
                                id="reportParameters.includeEpssProbability"
                                isChecked={formik.values.reportParameters.includeEpssProbability}
                                onChange={onChange}
                            />
                        )}
                        {isIncludeAdvisoryEnabled && (
                            <Checkbox
                                label="Include advisory"
                                id="reportParameters.includeAdvisory"
                                isChecked={formik.values.reportParameters.includeAdvisory}
                                onChange={onChange}
                            />
                        )}
                    </FormGroup>
                )}
                <FormLabelGroup
                    label="Configure collection included"
                    isRequired
                    fieldId="reportParameters.reportScope"
                    errors={formik.errors}
                >
                    <CollectionSelection
                        toggleId="reportParameters.reportScope"
                        id="reportParameters.reportScope"
                        selectedScope={formik.values.reportParameters.reportScope}
                        onChange={handleCollectionSelection('reportParameters.reportScope')}
                        onBlur={formik.handleBlur}
                        onValidateField={formik.validateField}
                    />
                </FormLabelGroup>
            </Form>
        </>
    );
}

export default ReportParametersForm;
