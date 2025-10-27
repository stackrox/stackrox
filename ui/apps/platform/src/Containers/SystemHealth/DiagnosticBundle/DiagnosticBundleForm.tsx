import { useEffect, useState } from 'react';
import type { ChangeEvent, MouseEvent as ReactMouseEvent, ReactElement } from 'react';
import {
    Checkbox,
    DatePicker,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    TimePicker,
} from '@patternfly/react-core';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';

import usePermissions from 'hooks/usePermissions';
import { fetchClusters } from 'services/ClustersService';
import type { DiagnosticBundleRequest } from 'services/DebugService';

export type DiagnosticBundleFormProps = {
    values: DiagnosticBundleRequest;
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
        setClusterSelectOpen(!clusterSelectOpen);
    }

    function onSelect(event: ReactMouseEvent | ChangeEvent, selection) {
        const newClusterFilter = values.filterByClusters.includes(selection)
            ? values.filterByClusters.filter((item) => item !== selection)
            : [...values.filterByClusters, selection];

        return setFieldValue('filterByClusters', newClusterFilter);
    }

    function clearSelection(e) {
        e.stopPropagation();

        setFieldValue('filterByClusters', []);
    }

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
                        variant="typeaheadmulti"
                        typeAheadAriaLabel="Type a cluster name"
                        onToggle={toggleClusterSelect}
                        onSelect={onSelect}
                        onClear={clearSelection}
                        selections={values.filterByClusters}
                        isOpen={clusterSelectOpen}
                        isDisabled={values.isDatabaseDiagnosticsOnly}
                    >
                        {availableClusterOptions.map((cluster) => (
                            <SelectOption key={cluster} value={cluster} />
                        ))}
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
