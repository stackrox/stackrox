import React, { ReactElement } from 'react';
import {
    DatePicker,
    Divider,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    PageSection,
    SelectOption,
    TextArea,
    TextInput,
    Title,
} from '@patternfly/react-core';

import {
    ReportFormValues,
    SetReportFormFieldValue,
} from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';
import usePermissions from 'hooks/usePermissions';
import { fixabilityLabels } from 'constants/reportConstants';
import {
    cvesDiscoveredSinceLabelMap,
    imageTypeLabelMap,
} from 'Containers/Vulnerabilities/VulnerablityReporting/utils';

import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import SelectSingle from 'Components/SelectSingle/SelectSingle';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import CollectionSelection from './CollectionSelection';

export type ReportParametersFormParams = {
    formValues: ReportFormValues;
    setFormFieldValue: SetReportFormFieldValue;
};

function ReportParametersForm({
    formValues,
    setFormFieldValue,
}: ReportParametersFormParams): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const canWriteCollections = hasReadWriteAccess('WorkflowAdministration');

    const handleTextChange = (fieldName: string) => (value: string) => {
        setFormFieldValue(fieldName, value);
    };

    const handleSelectChange = (name: string, value: string) => {
        setFormFieldValue(name, value);
    };

    const handleCheckboxSelectChange = (fieldName: string) => (selection: string[]) => {
        setFormFieldValue(fieldName, selection);
    };

    const handleDateSelection = (fieldName: string) => (_event, selection) => {
        setFormFieldValue(fieldName, selection);
    };

    const handleCollectionSelection = (fieldName: string) => (selection) => {
        setFormFieldValue(fieldName, selection);
    };

    const handleCVEsDiscoveredStartDate = handleDateSelection(
        'reportParameters.cvesDiscoveredStartDate'
    );

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Configure report parameters</Title>
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Form className="pf-u-py-lg pf-u-px-lg">
                <FormGroup label="Report name" isRequired fieldId="reportParameters.reportName">
                    <TextInput
                        isRequired
                        type="text"
                        id="reportName"
                        name="reportName"
                        value={formValues.reportParameters.reportName}
                        onChange={handleTextChange('reportParameters.reportName')}
                    />
                </FormGroup>
                <FormGroup label="Description" fieldId="reportParameters.description">
                    <TextArea
                        type="text"
                        id="description"
                        name="description"
                        value={formValues.reportParameters.description}
                        onChange={handleTextChange('reportParameters.description')}
                    />
                </FormGroup>
                <FormGroup label="CVE severity" isRequired fieldId="reportParameters.cveSeverities">
                    <CheckboxSelect
                        ariaLabel="CVE severity checkbox select"
                        selections={formValues.reportParameters.cveSeverities}
                        onChange={handleCheckboxSelectChange('reportParameters.cveSeverities')}
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
                </FormGroup>
                <FormGroup label="CVE status" isRequired fieldId="reportParameters.cveStatus">
                    <CheckboxSelect
                        ariaLabel="CVE status checkbox select"
                        selections={formValues.reportParameters.cveStatus}
                        onChange={handleCheckboxSelectChange('reportParameters.cveStatus')}
                        placeholderText="CVE status"
                    >
                        <SelectOption value="FIXABLE">{fixabilityLabels.FIXABLE}</SelectOption>
                        <SelectOption value="NOT_FIXABLE">
                            {fixabilityLabels.NOT_FIXABLE}
                        </SelectOption>
                    </CheckboxSelect>
                </FormGroup>
                <FormGroup label="Image type" isRequired fieldId="reportParameters.imageType">
                    <CheckboxSelect
                        ariaLabel="Image type checkbox select"
                        selections={formValues.reportParameters.imageType}
                        onChange={handleCheckboxSelectChange('reportParameters.imageType')}
                        placeholderText="Image type"
                    >
                        <SelectOption value="DEPLOYED">{imageTypeLabelMap.DEPLOYED}</SelectOption>
                        <SelectOption value="WATCHED">{imageTypeLabelMap.WATCHED}</SelectOption>
                    </CheckboxSelect>
                </FormGroup>
                <FormGroup
                    label="CVEs discovered since"
                    isRequired
                    fieldId="reportParameters.cvesDiscoveredSince"
                >
                    <SelectSingle
                        id="reportParameters.cvesDiscoveredSince"
                        value={formValues.reportParameters.cvesDiscoveredSince}
                        handleSelect={handleSelectChange}
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
                </FormGroup>
                {formValues.reportParameters.cvesDiscoveredSince === 'START_DATE' && (
                    <FormGroup isRequired fieldId="reportParameters.cvesDiscoveredStartDate">
                        <DatePicker
                            value={formValues.reportParameters.cvesDiscoveredStartDate}
                            onBlur={handleCVEsDiscoveredStartDate}
                            onChange={handleCVEsDiscoveredStartDate}
                        />
                    </FormGroup>
                )}
                <FormGroup isRequired fieldId="reportParameters.reportScope">
                    <CollectionSelection
                        selectedScope={formValues.reportParameters.reportScope}
                        initialReportScope={null}
                        onChange={handleCollectionSelection('reportParameters.reportScope')}
                        allowCreate={canWriteCollections}
                    />
                </FormGroup>
            </Form>
        </>
    );
}

export default ReportParametersForm;
