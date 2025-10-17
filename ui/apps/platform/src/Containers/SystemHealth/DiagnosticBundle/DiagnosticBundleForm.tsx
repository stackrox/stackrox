import { useEffect, useState } from 'react';
import type { ChangeEvent, MouseEvent as ReactMouseEvent, FormEvent, ReactElement } from 'react';
import {
    Form,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    TextInput,
} from '@patternfly/react-core';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';

import usePermissions from 'hooks/usePermissions';
import { fetchClusters } from 'services/ClustersService';
import type { DiagnosticBundleRequest } from 'services/DebugService';
import FilterByStartingTimeValidationMessage from './FilterByStartingTimeValidationMessage';

const startingTimeFormat = 'yyyy-mm-ddThh:mmZ'; // seconds are optional but UTC is required

export type DiagnosticBundleFormProps = {
    values: DiagnosticBundleRequest;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    setFieldValue: (field: string, value: any, shouldValidate?: boolean) => void;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    handleBlur: (e: any) => void;
    currentTimeObject: Date | null;
    isStartingTimeValid: boolean;
    startingTimeObject: Date | null;
    onChangeStartingTime: (event: FormEvent<HTMLInputElement>) => void;
};

function DiagnosticBundleForm({
    values,
    setFieldValue,
    handleBlur,
    currentTimeObject,
    isStartingTimeValid,
    startingTimeObject,
    onChangeStartingTime,
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

    function startingTimeChangeHandler(value: string, event: FormEvent<HTMLInputElement>) {
        onChangeStartingTime(event);
        return setFieldValue(event.currentTarget.id, value);
    }

    return (
        <Form>
            <p>You can filter which platform data to include in the Zip file (max size 50MB)</p>
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
            <FormGroup
                label="Filter by starting time"
                labelInfo={
                    <FilterByStartingTimeValidationMessage
                        currentTimeObject={currentTimeObject}
                        isStartingTimeValid={isStartingTimeValid}
                        startingTimeFormat={startingTimeFormat}
                        startingTimeObject={startingTimeObject}
                    />
                }
                fieldId="filterByStartingTime"
            >
                <TextInput
                    type="text"
                    id="filterByStartingTime"
                    placeholder={startingTimeFormat}
                    value={values.filterByStartingTime}
                    onChange={(event, value: string) => startingTimeChangeHandler(value, event)}
                    onBlur={handleBlur}
                />
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem>
                            To override default, use UTC format (seconds are optional)
                        </HelperTextItem>
                    </HelperText>
                </FormHelperText>
            </FormGroup>
        </Form>
    );
}

export default DiagnosticBundleForm;
