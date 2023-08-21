import React, { ReactElement } from 'react';
import {
    DatePicker,
    Divider,
    Flex,
    FlexItem,
    Form,
    PageSection,
    SelectOption,
    TextArea,
    TextInput,
    Title,
} from '@patternfly/react-core';
import { FormikProps } from 'formik';

import { ReportFormValues } from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';
import usePermissions from 'hooks/usePermissions';
import { fixabilityLabels } from 'constants/reportConstants';
import {
    cvesDiscoveredSinceLabelMap,
    imageTypeLabelMap,
} from 'Containers/Vulnerabilities/VulnerablityReporting/utils';

import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import SelectSingle from 'Components/SelectSingle/SelectSingle';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import CollectionSelection from './CollectionSelection';

export type ReportParametersFormParams = {
    title: string;
    formik: FormikProps<ReportFormValues>;
};

function ReportParametersForm({ title, formik }: ReportParametersFormParams): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const canWriteCollections = hasReadWriteAccess('WorkflowAdministration');

    const handleTextChange = (fieldName: string) => (value: string) => {
        formik.setFieldValue(fieldName, value);
    };

    const handleSelectChange = (name: string, value: string) => {
        formik.setFieldValue(name, value);
    };

    const handleCheckboxSelectChange = (fieldName: string) => (selection: string[]) => {
        formik.setFieldValue(fieldName, selection);
    };

    const handleDateSelection = (fieldName: string) => (_event, selection) => {
        formik.setFieldValue(fieldName, selection);
    };

    const handleCollectionSelection = (fieldName: string) => (selection) => {
        formik.setFieldValue(fieldName, selection);
    };

    const handleCVEsDiscoveredStartDate = handleDateSelection(
        'reportParameters.cvesDiscoveredStartDate'
    );

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">{title}</Title>
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Form className="pf-u-py-lg pf-u-px-lg">
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
                    label="Description"
                    fieldId="reportParameters.description"
                    errors={formik.errors}
                >
                    <TextArea
                        type="text"
                        id="reportParameters.description"
                        name="reportParameters.description"
                        value={formik.values.reportParameters.description}
                        onChange={handleTextChange('reportParameters.description')}
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
                    >
                        <SelectOption value="CRITICAL_VULNERABILITY_SEVERITY">
                            <Flex
                                className="pf-u-mx-sm"
                                spaceItems={{ default: 'spaceItemsSm' }}
                                alignItems={{ default: 'alignItemsCenter' }}
                            >
                                <VulnerabilitySeverityIconText severity="CRITICAL_VULNERABILITY_SEVERITY" />
                            </Flex>
                        </SelectOption>
                        <SelectOption value="IMPORTANT_VULNERABILITY_SEVERITY">
                            <Flex
                                className="pf-u-mx-sm"
                                spaceItems={{ default: 'spaceItemsSm' }}
                                alignItems={{ default: 'alignItemsCenter' }}
                            >
                                <VulnerabilitySeverityIconText severity="IMPORTANT_VULNERABILITY_SEVERITY" />
                            </Flex>
                        </SelectOption>
                        <SelectOption value="MODERATE_VULNERABILITY_SEVERITY">
                            <Flex
                                className="pf-u-mx-sm"
                                spaceItems={{ default: 'spaceItemsSm' }}
                                alignItems={{ default: 'alignItemsCenter' }}
                            >
                                <VulnerabilitySeverityIconText severity="MODERATE_VULNERABILITY_SEVERITY" />
                            </Flex>
                        </SelectOption>
                        <SelectOption value="LOW_VULNERABILITY_SEVERITY">
                            <Flex
                                className="pf-u-mx-sm"
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
                        handleSelect={handleSelectChange}
                        onBlur={formik.handleBlur}
                    >
                        <SelectOption
                            value="SINCE_LAST_REPORT"
                            description="Only applicable if there is a schedule configured in the report"
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
                <FormLabelGroup
                    label="Configure report scope"
                    isRequired
                    fieldId="reportParameters.reportScope"
                    errors={formik.errors}
                >
                    <CollectionSelection
                        toggleId="reportParameters.reportScope"
                        id="reportParameters.reportScope"
                        selectedScope={formik.values.reportParameters.reportScope}
                        onChange={handleCollectionSelection('reportParameters.reportScope')}
                        allowCreate={canWriteCollections}
                        onBlur={formik.handleBlur}
                        onValidateField={formik.validateField}
                    />
                </FormLabelGroup>
            </Form>
        </>
    );
}

export default ReportParametersForm;
