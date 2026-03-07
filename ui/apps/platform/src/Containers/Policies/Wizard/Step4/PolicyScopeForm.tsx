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

import useFeatureFlags from 'hooks/useFeatureFlags';
import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import type { ClientPolicy } from 'types/policy.proto';
import type { ListImage } from 'types/image.proto';
import { getImages } from 'services/imageService';

import { initialExcludedDeployment } from '../../policies.utils';
import PolicyScopeCardLegacy from './PolicyScopeCardLegacy';
import InclusionScopeCard from './InclusionScopeCard';

function PolicyScopeForm(): ReactElement {
    const [isExcludeImagesOpen, setIsExcludeImagesOpen] = useState(false);
    const [filterValue, setFilterValue] = useState('');
    const [images, setImages] = useState<ListImage[]>([]);
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const { clusters } = useFetchClustersForPermissions(['Deployment']);
    const { values, handleChange, setFieldValue } = useFormikContext<ClientPolicy>();
    const { scope, excludedDeploymentScopes, excludedImageNames } = values;

    const hasAuditLogEventSource = values.eventSource === 'AUDIT_LOG_EVENT';
    const hasNodeEventSource = values.eventSource === 'NODE_EVENT';
    const hasBuildLifecycle = values.lifecycleStages.includes('BUILD');
    const hasOnlyBuildLifecycle = values.lifecycleStages.length === 1 && hasBuildLifecycle;
    const filteredImages = filterValue
        ? images.filter((image) => image.name.toLowerCase().includes(filterValue.toLowerCase()))
        : images;

    const shouldShowCreateOption =
        filterValue && !filteredImages?.some((image) => image.name === filterValue);

    const hasResults = filteredImages?.length > 0 || shouldShowCreateOption;

    const isAllScopingDisabled = hasNodeEventSource;
    const isNewScopingEnabled = isFeatureFlagEnabled('ROX_LABEL_BASED_POLICY_SCOPING');

    function addNewScope() {
        setFieldValue('scope', [...scope, {}]);
    }

    function deleteScope(index: number) {
        const newScope = scope.filter((_, i) => i !== index);
        setFieldValue('scope', newScope);
    }

    function addNewExclusionDeploymentScope() {
        setFieldValue('excludedDeploymentScopes', [
            ...excludedDeploymentScopes,
            initialExcludedDeployment,
        ]);
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

    useEffect(() => {
        if (isNewScopingEnabled) {
            const nonEmptyScopes = scope.filter(
                (s) => s.cluster || s.namespace || s.label || s.clusterLabel || s.namespaceLabel
            );
            if (nonEmptyScopes.length !== scope.length) {
                setFieldValue('scope', nonEmptyScopes);
            }

            const nonEmptyExclusions = excludedDeploymentScopes.filter((e) => e.name || e.scope);
            if (nonEmptyExclusions.length !== excludedDeploymentScopes.length) {
                setFieldValue('excludedDeploymentScopes', nonEmptyExclusions);
            }
        }
        // Only run on mount
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // @TODO: Consider using a custom component for the multi-select typeahead dropdown. PolicyCategoriesSelectField.tsx is a good example too.
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
            <FlexItem flex={{ default: 'flex_1' }} className="pf-v5-u-p-lg">
                <Title headingLevel="h2">
                    {isNewScopingEnabled ? 'Coverage' : 'Scopes and Exclusions'}
                </Title>
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
                    title="Scopes and exclusions are not supported for policies which inspect node activity."
                    component="p"
                />
            )}
            {!isAllScopingDisabled && hasOnlyBuildLifecycle && (
                <Alert
                    className="pf-v5-u-mt-lg pf-v5-u-mx-lg"
                    isInline
                    variant="info"
                    title="Scopes are not supported for Build lifecycle stage."
                    component="p"
                />
            )}
            {!isAllScopingDisabled && !hasOnlyBuildLifecycle && (
                <>
                    <Flex direction={{ default: 'column' }} className="pf-v5-u-p-lg">
                        <Flex>
                            <FlexItem flex={{ default: 'flex_1' }}>
                                <Title headingLevel="h3">
                                    {isNewScopingEnabled ? 'Scope' : 'Restrict by scope'}
                                </Title>
                                <div className="pf-v5-u-mt-sm">
                                    {isNewScopingEnabled
                                        ? 'Apply this policy to one or more clusters, namespaces or deployments (if applicable). If no scopes are added, the policy will apply to all resources in your environment, except those excluded.'
                                        : 'Use Restrict by scope to enable this policy only for a specific cluster, namespace, or deployment label. You can add multiple scopes and also use regular expressions (RE2 syntax) for namespaces and deployment labels.'}
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
                                    {isNewScopingEnabled ? 'Add Scope' : 'Add inclusion scope'}
                                </Button>
                            </FlexItem>
                        </Flex>
                        <FlexItem>
                            <Grid hasGutter md={6} xl2={4}>
                                {scope?.map((_, index) => (
                                    // eslint-disable-next-line react/no-array-index-key
                                    <GridItem key={index}>
                                        {isNewScopingEnabled ? (
                                            <InclusionScopeCard
                                                index={index}
                                                scope={scope[index]}
                                                clusters={clusters}
                                                handleChange={handleChange}
                                                setFieldValue={setFieldValue}
                                                onDelete={() => deleteScope(index)}
                                                hasAuditLogEventSource={hasAuditLogEventSource}
                                            />
                                        ) : (
                                            <PolicyScopeCardLegacy
                                                type="inclusion"
                                                name={`scope[${index}]`}
                                                clusters={clusters}
                                                onDelete={() => deleteScope(index)}
                                                hasAuditLogEventSource={hasAuditLogEventSource}
                                            />
                                        )}
                                    </GridItem>
                                ))}
                            </Grid>
                        </FlexItem>
                    </Flex>
                    <Divider component="div" />
                    <Flex direction={{ default: 'column' }} className="pf-v5-u-p-lg">
                        <Flex>
                            <FlexItem flex={{ default: 'flex_1' }}>
                                <Title headingLevel="h3">
                                    {isNewScopingEnabled ? 'Exclusions' : 'Exclude by scope'}
                                </Title>
                                <div className="pf-v5-u-mt-sm">
                                    {isNewScopingEnabled
                                        ? 'Exclude one or more clusters, namespaces, or deployments (if applicable) when this policy is applied to workloads.'
                                        : 'Use Exclude by scope to exclude entities from your policy. This function is only available for Deploy and Runtime lifecycle stages. You can add multiple scopes and also use regular expressions (RE2 syntax) for namespaces and deployment labels.'}
                                </div>
                            </FlexItem>
                            <FlexItem
                                className="pf-v5-u-pr-md"
                                alignSelf={{ default: 'alignSelfCenter' }}
                            >
                                <Button
                                    variant="secondary"
                                    isDisabled={isAllScopingDisabled}
                                    onClick={addNewExclusionDeploymentScope}
                                >
                                    {isNewScopingEnabled ? 'Add Exclusion' : 'Add exclusion scope'}
                                </Button>
                            </FlexItem>
                        </Flex>
                        <FlexItem>
                            <Grid hasGutter md={6} xl={4}>
                                {excludedDeploymentScopes?.map((_, index) => (
                                    // eslint-disable-next-line react/no-array-index-key
                                    <GridItem key={index}>
                                        <PolicyScopeCardLegacy
                                            type="exclusion"
                                            name={`excludedDeploymentScopes[${index}]`}
                                            clusters={clusters}
                                            onDelete={() => deleteExclusion(index)}
                                            hasAuditLogEventSource={hasAuditLogEventSource}
                                            cardTitle={
                                                isNewScopingEnabled ? 'Exclusion' : undefined
                                            }
                                            showTooltips={isNewScopingEnabled}
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
                                {isNewScopingEnabled
                                    ? 'Exclude one or more images in a continuous integration system.'
                                    : "The exclude images setting only applies when you check images in a continuous integration system (the Build lifecycle stage). It won't have any effect if you use this policy to check running deployments (the Deploy lifecycle stage) or runtime activities (the Run lifecycle stage)."}
                            </div>
                        </FlexItem>
                        <FlexItem>
                            <FormGroup
                                label={
                                    isNewScopingEnabled
                                        ? 'Exclude images'
                                        : 'Exclude images (Build lifecycle only)'
                                }
                                fieldId="exclude-images"
                            >
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
                                            isDisabled={
                                                hasAuditLogEventSource || isAllScopingDisabled
                                            }
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
