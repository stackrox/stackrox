import { useEffect, useState } from 'react';
import type { MouseEvent as ReactMouseEvent, ReactElement, Ref } from 'react';
import {
    Button,
    Checkbox,
    Chip,
    ChipGroup,
    DatePicker,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    MenuToggle,
    Select,
    SelectList,
    SelectOption,
    TextInputGroup,
    TextInputGroupMain,
    TextInputGroupUtilities,
    TimePicker,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';
import { TimesIcon } from '@patternfly/react-icons';

import usePermissions from 'hooks/usePermissions';
import { fetchClusters } from 'services/ClustersService';
import { toggleItemInArray } from 'utils/arrayUtils';

export type DiagnosticBundleFormValues = {
    startingDate: string; // YYYY-MM-DD or empty string, derived from patternfly date picker
    startingTime: string; // HH:MM or empty string, derived from patternfly time picker
    filterByClusters: string[];
    isDatabaseDiagnosticsOnly: boolean;
    includeComplianceOperatorResources: boolean;
};

export type DiagnosticBundleFormProps = {
    values: DiagnosticBundleFormValues;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    setFieldValue: (field: string, value: any, shouldValidate?: boolean) => void;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    handleBlur: (e: any) => void;
};

function DiagnosticBundleForm({
    values,
    setFieldValue,
    handleBlur,
}: DiagnosticBundleFormProps): ReactElement {
    const [availableClusterOptions, setAvailableClusterOptions] = useState<string[]>([]);
    const [clusterSelectOpen, setClusterSelectOpen] = useState(false);
    const [inputValue, setInputValue] = useState('');
    const { hasReadAccess } = usePermissions();

    const hasReadAccessForCluster = hasReadAccess('Cluster');
    useEffect(() => {
        if (hasReadAccessForCluster) {
            fetchClusters()
                .then((clusters) => {
                    setAvailableClusterOptions(clusters.map(({ name }) => name));
                })
                .catch(() => {
                    // TODO display message when there is a place for minor errors
                });
        }
    }, [hasReadAccessForCluster]);

    function toggleClusterSelect() {
        setClusterSelectOpen((prev) => !prev);
    }

    function onSelect(_event: ReactMouseEvent | undefined, selection: string | number | undefined) {
        if (typeof selection === 'string') {
            const newClusterFilter = toggleItemInArray(values.filterByClusters, selection);
            setInputValue('');
            setFieldValue('filterByClusters', newClusterFilter);
        }
    }

    function clearSelection() {
        setInputValue('');
        setFieldValue('filterByClusters', []);
    }

    function onRemoveChip(clusterToRemove: string) {
        setFieldValue(
            'filterByClusters',
            values.filterByClusters.filter((cluster) => cluster !== clusterToRemove)
        );
    }

    const filteredClusters = availableClusterOptions.filter((cluster) =>
        cluster.toLowerCase().includes(inputValue.toLowerCase())
    );

    const toggle = (toggleRef: Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            variant="typeahead"
            onClick={toggleClusterSelect}
            isExpanded={clusterSelectOpen}
            isDisabled={values.isDatabaseDiagnosticsOnly}
            isFullWidth
        >
            <TextInputGroup isPlain isDisabled={values.isDatabaseDiagnosticsOnly}>
                <TextInputGroupMain
                    value={inputValue}
                    onClick={toggleClusterSelect}
                    onChange={(_event, value) => setInputValue(value)}
                    id="filterByClusters-input"
                    placeholder="Type a cluster name"
                    role="combobox"
                    isExpanded={clusterSelectOpen}
                    aria-controls="filterByClusters-listbox"
                >
                    <ChipGroup>
                        {values.filterByClusters.map((cluster) => (
                            <Chip
                                key={cluster}
                                onClick={(event: ReactMouseEvent) => {
                                    event.stopPropagation();
                                    onRemoveChip(cluster);
                                }}
                                aria-label={`Remove ${cluster}`}
                            >
                                {cluster}
                            </Chip>
                        ))}
                    </ChipGroup>
                </TextInputGroupMain>
                <TextInputGroupUtilities>
                    {values.filterByClusters.length > 0 && (
                        <Button
                            variant="plain"
                            onClick={clearSelection}
                            aria-label="Clear all selections"
                        >
                            <TimesIcon />
                        </Button>
                    )}
                </TextInputGroupUtilities>
            </TextInputGroup>
        </MenuToggle>
    );

    return (
        <Form>
            <FormGroup fieldId="diagnosticOptions">
                <Checkbox
                    label="Database diagnostics only"
                    id="isDatabaseDiagnosticsOnly"
                    isChecked={values.isDatabaseDiagnosticsOnly}
                    onChange={(_event, checked) =>
                        setFieldValue('isDatabaseDiagnosticsOnly', checked)
                    }
                    description="Only Central database, metrics, and logs. Other filters disabled."
                />
            </FormGroup>
            {hasReadAccessForCluster && (
                <FormGroup label="Filter by clusters" fieldId="filterByClusters">
                    <Select
                        id="filterByClusters"
                        isOpen={clusterSelectOpen}
                        selected={values.filterByClusters}
                        onSelect={onSelect}
                        onOpenChange={(isOpen) => setClusterSelectOpen(isOpen)}
                        toggle={toggle}
                    >
                        <SelectList
                            id="filterByClusters-listbox"
                            style={{ maxHeight: '300px', overflowY: 'auto' }}
                        >
                            {filteredClusters.length > 0 ? (
                                filteredClusters.map((cluster) => (
                                    <SelectOption key={cluster} value={cluster}>
                                        {cluster}
                                    </SelectOption>
                                ))
                            ) : (
                                <SelectOption isDisabled>No matching clusters</SelectOption>
                            )}
                        </SelectList>
                    </Select>
                    <FormHelperText>
                        <HelperText>
                            <HelperTextItem>
                                No clusters selected will include all clusters
                            </HelperTextItem>
                        </HelperText>
                    </FormHelperText>
                </FormGroup>
            )}
            <FormGroup label="Filter by starting time" fieldId="filterByStartingTime">
                <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                    <FlexItem>
                        <DatePicker
                            value={values.startingDate}
                            onChange={(_event, value) => setFieldValue('startingDate', value)}
                            isDisabled={values.isDatabaseDiagnosticsOnly}
                            inputProps={{
                                id: 'startingDate',
                                onBlur: handleBlur,
                            }}
                        />
                    </FlexItem>
                    <FlexItem>
                        <TimePicker
                            time={values.startingTime}
                            onChange={(_event, time) => setFieldValue('startingTime', time)}
                            is24Hour
                            isDisabled={values.isDatabaseDiagnosticsOnly}
                            inputProps={{
                                id: 'startingTime',
                                onBlur: handleBlur,
                            }}
                        />
                    </FlexItem>
                </Flex>
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem>
                            Default is 20 minutes ago. Override using the filters (UTC format)
                        </HelperTextItem>
                    </HelperText>
                </FormHelperText>
            </FormGroup>
            <FormGroup label="Additional diagnostics" fieldId="additionalDiagnostics">
                <Checkbox
                    label="Include compliance operator resources"
                    id="includeComplianceOperatorResources"
                    isChecked={values.includeComplianceOperatorResources}
                    isDisabled={values.isDatabaseDiagnosticsOnly}
                    onChange={(_event, checked) =>
                        setFieldValue('includeComplianceOperatorResources', checked)
                    }
                />
            </FormGroup>
        </Form>
    );
}

export default DiagnosticBundleForm;
