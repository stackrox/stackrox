import { useEffect, useState } from 'react';
import type { MouseEvent as ReactMouseEvent, ReactElement, Ref } from 'react';
import { useFormikContext } from 'formik';
import {
    Alert,
    Button,
    Chip,
    ChipGroup,
    Divider,
    Flex,
    FlexItem,
    FormGroup,
    Grid,
    GridItem,
    MenuToggle,
    Select,
    SelectList,
    SelectOption,
    TextInputGroup,
    TextInputGroupMain,
    TextInputGroupUtilities,
    Title,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';
import { TimesIcon } from '@patternfly/react-icons';

import type { ClientPolicy } from 'types/policy.proto';
import type { ListImage } from 'types/image.proto';
import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import { getImages } from 'services/imageService';
import PolicyScopeCard from './PolicyScopeCard';

function PolicyScopeForm(): ReactElement {
    const [isExcludeImagesOpen, setIsExcludeImagesOpen] = useState(false);
    const [filterValue, setFilterValue] = useState('');
    const [images, setImages] = useState<ListImage[]>([]);
    const { clusters } = useFetchClustersForPermissions(['Deployment']);
    const { values, setFieldValue } = useFormikContext<ClientPolicy>();
    const { scope, excludedDeploymentScopes, excludedImageNames } = values;

    const hasAuditLogEventSource = values.eventSource === 'AUDIT_LOG_EVENT';
    const hasNodeEventSource = values.eventSource === 'NODE_EVENT';
    const hasBuildLifecycle = values.lifecycleStages.includes('BUILD');
    const hasOnlyBuildLifecycle = values.lifecycleStages.length === 1 && hasBuildLifecycle;
    // Filter images based on the current filter value
    const filteredImages = filterValue
        ? images.filter((image) => image.name.toLowerCase().includes(filterValue.toLowerCase()))
        : images;

    const shouldShowCreateOption =
        filterValue && !filteredImages?.some((image) => image.name === filterValue);

    // Check if we have any content to show
    const hasResults = filteredImages?.length > 0 || shouldShowCreateOption;

    const isAllScopingDisabled = hasNodeEventSource;

    function addNewScope() {
        setFieldValue('scope', [...scope, {}]);
    }

    function deleteScope(index: number) {
        const newScope = scope.filter((_, i) => i !== index);
        setFieldValue('scope', newScope);
    }

    function addNewExclusion() {
        setFieldValue('excludedDeploymentScopes', [...excludedDeploymentScopes, {}]);
    }

    function deleteExclusion(index: number) {
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
    }, []);

    // @TODO: Consider using a custom component for the multi-select typeahead dropdown. PolicyCategoriesSelectField.tsx is a good example too.
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
            <FlexItem flex={{ default: 'flex_1' }} className="pf-v5-u-p-lg">
                <Title headingLevel="h2">Scopes and Exclusions</Title>
                <div className="pf-v5-u-mt-sm">
                    Configure the resources to be applied to, or excluded from this policy.
                </div>
            </FlexItem>
            <Divider component="div" />
            {isAllScopingDisabled && (
                <Alert
                    className="pf-v5-u-mt-lg pf-v5-u-mx-lg"
                    isInline
                    variant="info"
                    title="Scopes and exclusions are not supported for policies on node events."
                    component="p"
                />
            )}
            {!isAllScopingDisabled && hasOnlyBuildLifecycle && (
                <Alert
                    className="pf-v5-u-mt-lg pf-v5-u-mx-lg"
                    isInline
                    variant="info"
                    title="Scopes are not required in the Build lifecycle stage."
                    component="p"
                />
            )}
            {!isAllScopingDisabled && !hasOnlyBuildLifecycle && (
                <>
                    <Flex direction={{ default: 'column' }} className="pf-v5-u-p-lg">
                        <Flex>
                            <FlexItem flex={{ default: 'flex_1' }}>
                                <Title headingLevel="h3">Scope</Title>
                                <div className="pf-v5-u-mt-sm">
                                    Apply this policy to one or more clusters, namespaces or
                                    deployments (if applicable). If no scopes are added, the policy
                                    will apply to all resources in your environment, except those
                                    excluded.
                                </div>
                            </FlexItem>
                            <FlexItem
                                className="pf-v5-u-pr-md"
                                alignSelf={{ default: 'alignSelfCenter' }}
                            >
                                <Button
                                    variant="secondary"
                                    onClick={addNewScope}
                                    isDisabled={isAllScopingDisabled}
                                >
                                    Add Scope
                                </Button>
                            </FlexItem>
                        </Flex>
                        <FlexItem>
                            <Grid hasGutter md={6} xl={4}>
                                {scope?.map((s, index) => (
                                    // eslint-disable-next-line react/no-array-index-key
                                    <GridItem key={index}>
                                        <PolicyScopeCard
                                            type="scope"
                                            name={`scope[${index}]`}
                                            clusters={clusters}
                                            onDelete={() => deleteScope(index)}
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
                                <Title headingLevel="h3">Exclusions</Title>
                                <div className="pf-v5-u-mt-sm">
                                    Exclude one or more clusters, namespaces, or deployments (if
                                    applicable) when this policy is applied to workloads.
                                </div>
                            </FlexItem>
                            <FlexItem
                                className="pf-v5-u-pr-md"
                                alignSelf={{ default: 'alignSelfCenter' }}
                            >
                                <Button
                                    variant="secondary"
                                    isDisabled={isAllScopingDisabled}
                                    onClick={addNewExclusion}
                                >
                                    Add Exclusion
                                </Button>
                            </FlexItem>
                        </Flex>
                        <FlexItem>
                            <Grid hasGutter md={6} xl={4}>
                                {excludedDeploymentScopes?.map((s, index) => (
                                    // eslint-disable-next-line react/no-array-index-key
                                    <GridItem key={index}>
                                        <PolicyScopeCard
                                            type="exclusion"
                                            name={`excludedDeploymentScopes[${index}]`}
                                            clusters={clusters}
                                            onDelete={() => deleteExclusion(index)}
                                            hasAuditLogEventSource={hasAuditLogEventSource}
                                        />
                                    </GridItem>
                                ))}
                            </Grid>
                        </FlexItem>
                    </Flex>
                </>
            )}
            {hasBuildLifecycle && (
                <>
                    <Divider component="div" />
                    <Flex direction={{ default: 'column' }} className="pf-v5-u-p-lg">
                        <FlexItem flex={{ default: 'flex_1' }}>
                            <Title headingLevel="h3">Exclude images</Title>
                            <div className="pf-v5-u-mt-sm">
                                Exclude one or more images when this policy is evaluated in a
                                continuous integration system.
                            </div>
                        </FlexItem>
                        <FlexItem>
                            <FormGroup label="Exclude images" fieldId="exclude-images">
                                <Select
                                    isOpen={isExcludeImagesOpen}
                                    selected={excludedImageNames}
                                    onSelect={handleChangeMultiSelect}
                                    onOpenChange={(nextOpen: boolean) =>
                                        setIsExcludeImagesOpen(nextOpen)
                                    }
                                    toggle={(toggleRef: Ref<MenuToggleElement>) => (
                                        <MenuToggle
                                            variant="typeahead"
                                            aria-label="Typeahead menu toggle"
                                            onClick={() => setIsExcludeImagesOpen((prev) => !prev)}
                                            innerRef={toggleRef}
                                            isExpanded={isExcludeImagesOpen}
                                            isDisabled={isAllScopingDisabled}
                                            className="pf-v5-u-w-100"
                                        >
                                            <TextInputGroup isPlain>
                                                <TextInputGroupMain
                                                    value={filterValue}
                                                    onClick={() =>
                                                        setIsExcludeImagesOpen((prev) => !prev)
                                                    }
                                                    onChange={(_event, value) =>
                                                        setFilterValue(value)
                                                    }
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
                                                                setFieldValue(
                                                                    'excludedImageNames',
                                                                    []
                                                                );
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
                                                        isSelected={excludedImageNames.includes(
                                                            image.name
                                                        )}
                                                    >
                                                        {image.name}
                                                    </SelectOption>
                                                ))}
                                                {shouldShowCreateOption && (
                                                    <SelectOption
                                                        key={`create-${filterValue}`}
                                                        value={filterValue}
                                                    >
                                                        Create exclusion for images starting with
                                                        &quot;
                                                        {filterValue}&quot;
                                                    </SelectOption>
                                                )}
                                            </>
                                        ) : (
                                            <SelectOption isDisabled>
                                                {filterValue
                                                    ? 'No images found'
                                                    : 'No images available'}
                                            </SelectOption>
                                        )}
                                    </SelectList>
                                </Select>
                            </FormGroup>
                        </FlexItem>
                    </Flex>
                </>
            )}
        </Flex>
    );
}

export default PolicyScopeForm;
