/* eslint-disable @typescript-eslint/no-unsafe-return */
import React, { useEffect, useState, ReactElement, FormEvent } from 'react';
import {
    Form,
    FormGroup,
    Select,
    SelectOption,
    SelectVariant,
    TextInput,
} from '@patternfly/react-core';

import usePermissions from 'hooks/usePermissions';
import { fetchClusters } from 'services/ClustersService';
import { DiagnosticBundleRequest } from 'services/DebugService';
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

    function onSelect(event: React.MouseEvent | React.ChangeEvent, selection) {
        const newClusterFilter = values.filterByClusters.includes(selection)
            ? values.filterByClusters.filter((item) => item !== selection)
            : [...values.filterByClusters, selection];

        return setFieldValue('filterByClusters', newClusterFilter);
    }

    function clearSelection(e) {
        e.stopPropagation();

        setFieldValue('filterByClusters', []);
    }

    function startingTimeChangeHandler(value: string, event: React.FormEvent<HTMLInputElement>) {
        onChangeStartingTime(event);
        return setFieldValue(event.currentTarget.id, value);
    }

    return (
        <Form>
            <p>You can filter which platform data to include in the Zip file (max size 50MB)</p>
            {hasReadAccessForCluster && (
                <FormGroup
                    label="Filter by clusters"
                    fieldId="filterByClusters"
                    helperText="No clusters selected will include all clusters"
                >
                    <Select
                        id="filterByClusters"
                        variant={SelectVariant.typeaheadMulti}
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
                helperText="To override default, use UTC format (seconds are optional)"
            >
                <TextInput
                    type="text"
                    id="filterByStartingTime"
                    placeholder={startingTimeFormat}
                    value={values.filterByStartingTime}
                    onChange={startingTimeChangeHandler}
                    onBlur={handleBlur}
                />
            </FormGroup>
        </Form>
    );
}

export default DiagnosticBundleForm;
