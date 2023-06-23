import React, { ReactElement } from 'react';
import {
    DatePicker,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    SelectOption,
    TextArea,
    TextInput,
} from '@patternfly/react-core';
import set from 'lodash/set';

import {
    ReportFormValues,
    SetReportFormValues,
} from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';

import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import SelectSingle from 'Components/SelectSingle/SelectSingle';

export type ReportParametersFormParams = {
    formValues: ReportFormValues;
    setFormValues: SetReportFormValues;
};

const CriticalSeverityIcon = SeverityIcons.CRITICAL_VULNERABILITY_SEVERITY;
const ImportantSeverityIcon = SeverityIcons.IMPORTANT_VULNERABILITY_SEVERITY;
const ModerateSeverityIcon = SeverityIcons.MODERATE_VULNERABILITY_SEVERITY;
const LowSeverityIcon = SeverityIcons.LOW_VULNERABILITY_SEVERITY;

function ReportParametersForm({
    formValues,
    setFormValues,
}: ReportParametersFormParams): ReactElement {
    const handleTextChange = (fieldName: string) => (value: string) => {
        setFormValues((prevValues) => {
            const newValues = { ...prevValues };
            set(newValues, fieldName, value);
            return newValues;
        });
    };

    const handleSelectChange = (name: string, value: string) => {
        setFormValues((prevValues) => {
            const newValues = { ...prevValues };
            set(newValues, name, value);
            return newValues;
        });
    };

    const handleCheckboxSelectChange = (fieldName: string) => (selection: string[]) => {
        setFormValues((prevValues) => {
            const newValues = { ...prevValues };
            set(newValues, fieldName, selection);
            return newValues;
        });
    };

    const handleDateSelection = (fieldName: string) => (_event, str) => {
        setFormValues((prevValues) => {
            const newValues = { ...prevValues };
            set(newValues, fieldName, str);
            return newValues;
        });
    };

    const handleCVEsDiscoveredStartDate = handleDateSelection(
        'reportParameters.cvesDiscoveredStartDate'
    );

    return (
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
                            <FlexItem>
                                <CriticalSeverityIcon />
                            </FlexItem>
                            <FlexItem>Critical</FlexItem>
                        </Flex>
                    </SelectOption>
                    <SelectOption value="IMPORTANT_VULNERABILITY_SEVERITY">
                        <Flex
                            className="pf-u-mx-sm"
                            spaceItems={{ default: 'spaceItemsSm' }}
                            alignItems={{ default: 'alignItemsCenter' }}
                        >
                            <FlexItem>
                                <ImportantSeverityIcon />
                            </FlexItem>
                            <FlexItem>Important</FlexItem>
                        </Flex>
                    </SelectOption>
                    <SelectOption value="MODERATE_VULNERABILITY_SEVERITY">
                        <Flex
                            className="pf-u-mx-sm"
                            spaceItems={{ default: 'spaceItemsSm' }}
                            alignItems={{ default: 'alignItemsCenter' }}
                        >
                            <FlexItem>
                                <ModerateSeverityIcon />
                            </FlexItem>
                            <FlexItem>Moderate</FlexItem>
                        </Flex>
                    </SelectOption>
                    <SelectOption value="LOW_VULNERABILITY_SEVERITY">
                        <Flex
                            className="pf-u-mx-sm"
                            spaceItems={{ default: 'spaceItemsSm' }}
                            alignItems={{ default: 'alignItemsCenter' }}
                        >
                            <FlexItem>
                                <LowSeverityIcon />
                            </FlexItem>
                            <FlexItem>Low</FlexItem>
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
                    <SelectOption value="FIXABLE">Fixable</SelectOption>
                    <SelectOption value="NOT_FIXABLE">Not fixable</SelectOption>
                </CheckboxSelect>
            </FormGroup>
            <FormGroup label="Image type" isRequired fieldId="reportParameters.imageType">
                <CheckboxSelect
                    ariaLabel="Image type checkbox select"
                    selections={formValues.reportParameters.imageType}
                    onChange={handleCheckboxSelectChange('reportParameters.imageType')}
                    placeholderText="Image type"
                >
                    <SelectOption value="DEPLOYED">Deployed images</SelectOption>
                    <SelectOption value="WATCHED">Watched images</SelectOption>
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
                        Last successful scheduled run report
                    </SelectOption>
                    <SelectOption
                        value="START_DATE"
                        description="Custom start date for the discovered CVE that were run on-demand or downloaded"
                    >
                        Custom start date
                    </SelectOption>
                    <SelectOption
                        value="ALL_VULN"
                        description="Show all detected CVEs from the beginning of cluster setup"
                    >
                        All time
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
        </Form>
    );
}

export default ReportParametersForm;
