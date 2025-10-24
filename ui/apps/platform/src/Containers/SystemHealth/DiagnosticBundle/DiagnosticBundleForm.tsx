import { useEffect, useState } from 'react';
import type { MouseEvent as ReactMouseEvent, FormEvent, ReactElement, Ref } from 'react';
import {
    Button,
    Chip,
    ChipGroup,
    Form,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    MenuToggle,
    Select,
    SelectList,
    SelectOption,
    TextInput,
    TextInputGroup,
    TextInputGroupMain,
    TextInputGroupUtilities,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';
import { TimesIcon } from '@patternfly/react-icons';

import usePermissions from 'hooks/usePermissions';
import { fetchClusters } from 'services/ClustersService';
import type { DiagnosticBundleRequest } from 'services/DebugService';
import FilterByStartingTimeValidationMessage from './FilterByStartingTimeValidationMessage';
import { toggleItemInArray } from 'utils/arrayUtils';

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
        setClusterSelectOpen(!clusterSelectOpen);
    }

    function onSelect(_event: ReactMouseEvent | undefined, selection: string | number | undefined) {
        if (typeof selection === 'string') {
            const newClusterFilter = toggleItemInArray(values.filterByClusters, selection);
            setInputValue('');
            return setFieldValue('filterByClusters', newClusterFilter);
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

    function startingTimeChangeHandler(value: string, event: FormEvent<HTMLInputElement>) {
        onChangeStartingTime(event);
        return setFieldValue(event.currentTarget.id, value);
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
            isFullWidth
        >
            <TextInputGroup isPlain>
                <TextInputGroupMain
                    value={inputValue}
                    onClick={toggleClusterSelect}
                    onChange={(_event, value) => setInputValue(value)}
                    id="filterByClusters-input"
                    placeholder="Type a cluster name"
                    role="combobox"
                    isExpanded={clusterSelectOpen}
                    aria-controls="filterByClusters-listbox"
                    style={{ maxWidth: '600px' }}
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
            <p>You can filter which platform data to include in the Zip file (max size 50MB)</p>
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
