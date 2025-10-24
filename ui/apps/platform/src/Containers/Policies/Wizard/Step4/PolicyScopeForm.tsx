import React, { useEffect, useState } from 'react';
import type { MouseEvent as ReactMouseEvent, ReactElement, Ref } from 'react';
import { useFormikContext } from 'formik';
import {
    Flex,
    FlexItem,
    Title,
    Button,
    Divider,
    Grid,
    GridItem,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    Select,
    SelectOption,
    SelectList,
    MenuToggle,
    TextInputGroup,
    TextInputGroupMain,
    TextInputGroupUtilities,
    ChipGroup,
    Chip,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';
import { TimesIcon } from '@patternfly/react-icons';

import type { ClientPolicy } from 'types/policy.proto';
import type { ListImage } from 'types/image.proto';
import type { ListDeployment } from 'types/deployment.proto';
import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import { getImages } from 'services/imageService';
import { fetchDeploymentsWithProcessInfoLegacy as fetchDeploymentsWithProcessInfo } from 'services/DeploymentsService';
import PolicyScopeCard from './PolicyScopeCard';

function PolicyScopeForm(): ReactElement {
    const [isExcludeImagesOpen, setIsExcludeImagesOpen] = useState(false);
    const [filterValue, setFilterValue] = useState('');
    const [images, setImages] = useState<ListImage[]>([]);
    const [deployments, setDeployments] = useState<ListDeployment[]>([]);
    const { clusters } = useFetchClustersForPermissions(['Deployment']);
    const { values, setFieldValue } = useFormikContext<ClientPolicy>();
    const { scope, excludedDeploymentScopes, excludedImageNames } = values;

    const hasAuditLogEventSource = values.eventSource === 'AUDIT_LOG_EVENT';
    const hasBuildLifecycle = values.lifecycleStages.includes('BUILD');
    const hasDeployOrRuntimeLifecycle =
        values.lifecycleStages.includes('DEPLOY') || values.lifecycleStages.includes('RUNTIME');

    // Filter images based on the current filter value
    const filteredImages = filterValue
        ? images.filter((image) => image.name.toLowerCase().includes(filterValue.toLowerCase()))
        : images;

    const shouldShowCreateOption =
        filterValue && !filteredImages?.some((image) => image.name === filterValue);

    // Check if we have any content to show
    const hasResults = filteredImages?.length > 0 || shouldShowCreateOption;

    function addNewInclusionScope() {
        setFieldValue('scope', [...scope, {}]);
    }

    function deleteInclusionScope(index: number) {
        const newScope = scope.filter((_, i) => i !== index);
        setFieldValue('scope', newScope);
    }

    function addNewExclusionDeploymentScope() {
        setFieldValue('excludedDeploymentScopes', [...excludedDeploymentScopes, {}]);
    }

    function deleteExclusionDeploymentScope(index: number) {
        const newScope = excludedDeploymentScopes.filter((_, i) => i !== index);
        setFieldValue('excludedDeploymentScopes', newScope);
    }

    function handleChangeMultiSelect(
        _event: ReactMouseEvent | undefined,
        selectedImage: string | number | undefined
    ) {
        if (!selectedImage || typeof selectedImage === 'number') {
            return;
        }

        if (excludedImageNames.includes(selectedImage)) {
            const newExclusions = excludedImageNames.filter((image) => image !== selectedImage);
            setFieldValue('excludedImageNames', newExclusions);
        } else {
            setFieldValue('excludedImageNames', [...excludedImageNames, selectedImage]);
        }
        setFilterValue('');
    }

    useEffect(() => {
        getImages()
            .then((response) => {
                setImages(response);
            })
            .catch(() => {
                // TODO
            });

        // TODO from ROX-14643 and stackrox/stackrox/issues/2725
        // Move request to exclusion card to add restSearch for cluster or namespace if specified in exclusion scope.
        // Search element to support creatable deployment names.
        const restSort = { field: 'Deployment', reversed: false }; // ascending by name
        fetchDeploymentsWithProcessInfo([], restSort, 0, 0)
            .then((response) => {
                const deploymentList = response
                    .map(({ deployment }) => deployment)
                    .filter(({ name }, i, array) => i === 0 || name !== array[i - 1].name);
                setDeployments(deploymentList);
            })
            .catch(() => {
                // TODO
            });
    }, []);

    // @TODO: Consider using a custom component for the multi-select typeahead dropdown. PolicyCategoriesSelectField.tsx is a good example too.
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
            <FlexItem flex={{ default: 'flex_1' }} className="pf-v5-u-p-lg">
                <Title headingLevel="h2">Scope</Title>
                <div className="pf-v5-u-mt-sm">
                    Create scopes to restrict or exclude your policy from entities within your
                    environment.
                </div>
            </FlexItem>
            <Divider component="div" />
            <Flex direction={{ default: 'column' }} className="pf-v5-u-p-lg">
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h3">Restrict by scope</Title>
                        <div className="pf-v5-u-mt-sm">
                            Use Restrict by scope to enable this policy only for a specific cluster,
                            namespace, or deployment label. You can add multiple scopes and also use
                            regular expressions (RE2 syntax) for namespaces and deployment labels.
                        </div>
                    </FlexItem>
                    <FlexItem className="pf-v5-u-pr-md" alignSelf={{ default: 'alignSelfCenter' }}>
                        <Button variant="secondary" onClick={addNewInclusionScope}>
                            Add inclusion scope
                        </Button>
                    </FlexItem>
                </Flex>
                <FlexItem>
                    <Grid hasGutter md={6} xl={4}>
                        {scope?.map((_, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index}>
                                <PolicyScopeCard
                                    type="inclusion"
                                    name={`scope[${index}]`}
                                    clusters={clusters}
                                    onDelete={() => deleteInclusionScope(index)}
                                    hasAuditLogEventSource={hasAuditLogEventSource}
                                />
                            </GridItem>
                        ))}
                    </Grid>
                </FlexItem>
            </Flex>
            <Divider component="div" />
            <Flex direction={{ default: 'column' }} className="pf-v5-u-p-lg">
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h3">Exclude by scope</Title>
                        <div className="pf-v5-u-mt-sm">
                            Use Exclude by scope to exclude entities from your policy. This function
                            is only available for Deploy and Runtime lifecycle stages. You can add
                            multiple scopes and also use regular expressions (RE2 syntax) for
                            namespaces and deployment labels.
                        </div>
                    </FlexItem>
                    <FlexItem className="pf-v5-u-pr-md" alignSelf={{ default: 'alignSelfCenter' }}>
                        <Button
                            variant="secondary"
                            isDisabled={!hasDeployOrRuntimeLifecycle}
                            onClick={addNewExclusionDeploymentScope}
                        >
                            Add exclusion scope
                        </Button>
                    </FlexItem>
                </Flex>
                <FlexItem>
                    <Grid hasGutter md={6} xl={4}>
                        {excludedDeploymentScopes?.map((_, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index}>
                                <PolicyScopeCard
                                    type="exclusion"
                                    name={`excludedDeploymentScopes[${index}]`}
                                    clusters={clusters}
                                    deployments={deployments}
                                    onDelete={() => deleteExclusionDeploymentScope(index)}
                                    hasAuditLogEventSource={hasAuditLogEventSource}
                                />
                            </GridItem>
                        ))}
                    </Grid>
                </FlexItem>
            </Flex>
            <Divider component="div" />
            <Flex direction={{ default: 'column' }} className="pf-v5-u-p-lg">
                <FlexItem flex={{ default: 'flex_1' }}>
                    <Title headingLevel="h3">Exclude images</Title>
                    <div className="pf-v5-u-mt-sm">
                        The exclude images setting only applies when you check images in a
                        continuous integration system (the Build lifecycle stage). It won&apos;t
                        have any effect if you use this policy to check running deployments (the
                        Deploy lifecycle stage) or runtime activities (the Run lifecycle stage).
                    </div>
                </FlexItem>
                <FlexItem>
                    <FormGroup
                        label="Exclude images (Build lifecycle only)"
                        fieldId="exclude-images"
                    >
                        <Select
                            isOpen={isExcludeImagesOpen}
                            selected={excludedImageNames}
                            onSelect={handleChangeMultiSelect}
                            onOpenChange={(nextOpen: boolean) => setIsExcludeImagesOpen(nextOpen)}
                            toggle={(toggleRef: Ref<MenuToggleElement>) => (
                                <MenuToggle
                                    variant="typeahead"
                                    aria-label="Typeahead menu toggle"
                                    onClick={() => setIsExcludeImagesOpen(!isExcludeImagesOpen)}
                                    innerRef={toggleRef}
                                    isExpanded={isExcludeImagesOpen}
                                    isDisabled={hasAuditLogEventSource || !hasBuildLifecycle}
                                    className="pf-v5-u-w-100"
                                >
                                    <TextInputGroup isPlain>
                                        <TextInputGroupMain
                                            value={filterValue}
                                            onClick={() =>
                                                setIsExcludeImagesOpen(!isExcludeImagesOpen)
                                            }
                                            onChange={(_event, value) => setFilterValue(value)}
                                            autoComplete="off"
                                            placeholder="Select images to exclude"
                                        >
                                            {excludedImageNames.length > 0 && (
                                                <ChipGroup>
                                                    {excludedImageNames.map((image) => (
                                                        <Chip
                                                            key={image}
                                                            onClick={(event) => {
                                                                event.stopPropagation();
                                                                handleChangeMultiSelect(
                                                                    event,
                                                                    image
                                                                );
                                                            }}
                                                        >
                                                            {image}
                                                        </Chip>
                                                    ))}
                                                </ChipGroup>
                                            )}
                                        </TextInputGroupMain>
                                        <TextInputGroupUtilities>
                                            {excludedImageNames.length > 0 && (
                                                <Button
                                                    variant="plain"
                                                    onClick={(event) => {
                                                        event.stopPropagation();
                                                        setFieldValue('excludedImageNames', []);
                                                        setFilterValue('');
                                                    }}
                                                    aria-label="Clear input value"
                                                >
                                                    <TimesIcon />
                                                </Button>
                                            )}
                                        </TextInputGroupUtilities>
                                    </TextInputGroup>
                                </MenuToggle>
                            )}
                        >
                            <SelectList>
                                {hasResults ? (
                                    <>
                                        {filteredImages?.map((image) => (
                                            <SelectOption
                                                key={image.name}
                                                value={image.name}
                                                isSelected={excludedImageNames.includes(image.name)}
                                            >
                                                {image.name}
                                            </SelectOption>
                                        ))}
                                        {shouldShowCreateOption && (
                                            <SelectOption
                                                key={`create-${filterValue}`}
                                                value={filterValue}
                                            >
                                                Create exclusion for images starting with &quot;
                                                {filterValue}&quot;
                                            </SelectOption>
                                        )}
                                    </>
                                ) : (
                                    <SelectOption isDisabled>
                                        {filterValue ? 'No images found' : 'No images available'}
                                    </SelectOption>
                                )}
                            </SelectList>
                        </Select>
                        <FormHelperText>
                            <HelperText>
                                <HelperTextItem>
                                    Select all images from the list for which you don&apos;t want to
                                    trigger a violation for the policy.
                                </HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    </FormGroup>
                </FlexItem>
            </Flex>
        </Flex>
    );
}

export default PolicyScopeForm;
