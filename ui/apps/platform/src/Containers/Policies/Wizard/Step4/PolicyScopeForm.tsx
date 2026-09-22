import { useEffect, useState } from 'react';
import type { MouseEvent as ReactMouseEvent, ReactElement, Ref } from 'react';
import { useFormikContext } from 'formik';
import {
    Alert,
    Button,
    Checkbox,
    Divider,
    Flex,
    FlexItem,
    FormGroup,
    FormHelperText,
    Grid,
    GridItem,
    HelperText,
    HelperTextItem,
    Label,
    LabelGroup,
    MenuToggle,
    Select,
    SelectList,
    SelectOption,
    Stack,
    TextInputGroup,
    TextInputGroupMain,
    TextInputGroupUtilities,
    Title,
} from '@patternfly/react-core';

import type { MenuToggleElement } from '@patternfly/react-core';
import { TimesIcon } from '@patternfly/react-icons';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import { policyWorkloadTypeLabels, policyWorkloadTypes } from 'types/policy.proto';
import type { ClientPolicy, PolicyWorkloadType } from 'types/policy.proto';
import type { ListImage } from 'types/image.proto';
import { getImages } from 'services/imageService';
import { toggleItemInArray } from 'utils/arrayUtils';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';

import {
    formatWorkloadTypeExclusionMessage,
    initialExcludedDeployment,
    initialScope,
} from '../../policies.utils';
import PolicyScopeCardLegacy from './PolicyScopeCardLegacy';
import InclusionScopeCard from './InclusionScopeCard';
import ExclusionScopeCard from './ExclusionScopeCard';

function PolicyScopeRE2Description(): ReactElement {
    return (
        <div>
            Every field except Cluster can use RE2 matching. Empty fields apply to all values (no
            filter).{' '}
            <ExternalLink>
                <a
                    href="https://github.com/google/re2/wiki/syntax"
                    target="_blank"
                    rel="noopener noreferrer"
                >
                    Learn how to use regex here
                </a>
            </ExternalLink>
        </div>
    );
}

function PolicyScopeForm(): ReactElement {
    const [isExcludeImagesOpen, setIsExcludeImagesOpen] = useState(false);
    const [filterValue, setFilterValue] = useState('');
    const [images, setImages] = useState<ListImage[]>([]);
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const { clusters } = useFetchClustersForPermissions(['Deployment']);
    const { values, handleChange, setFieldValue } = useFormikContext<ClientPolicy>();
    const { scope, excludedDeploymentScopes, excludedImageNames } = values;
    const excludedWorkloadTypes = values.excludedWorkloadTypes ?? [];

    const showWorkloadTypeExclusion = isFeatureFlagEnabled('ROX_POLICY_WORKLOAD_TYPE_EXCLUSION');

    const hasAuditLogEventSource = values.eventSource === 'AUDIT_LOG_EVENT';
    const hasNodeEventSource = values.eventSource === 'NODE_EVENT';
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

    const isAllScopingDisabled = hasNodeEventSource;
    const isWorkloadTypeExclusionDisabled =
        isAllScopingDisabled || hasAuditLogEventSource || !hasDeployOrRuntimeLifecycle;

    function getWorkloadTypeExclusionInfo(): string {
        if (hasNodeEventSource) {
            return 'The selected event source does not support resource targeting.';
        }
        if (hasAuditLogEventSource) {
            return 'Workload type exclusions are not available for audit log event sources.';
        }
        if (!hasDeployOrRuntimeLifecycle) {
            return 'Workload type exclusions require the Deploy or Runtime lifecycle stage.';
        }
        return formatWorkloadTypeExclusionMessage(excludedWorkloadTypes);
    }

    function addNewInclusionScope() {
        setFieldValue('scope', [...scope, initialScope]);
    }

    function deleteInclusionScope(index: number) {
        const newScope = scope.filter((_, i) => i !== index);
        setFieldValue('scope', newScope);
    }

    function addNewExclusionDeploymentScope() {
        setFieldValue('excludedDeploymentScopes', [
            ...excludedDeploymentScopes,
            initialExcludedDeployment,
        ]);
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

    function handleChangeWorkloadType(type: PolicyWorkloadType) {
        setFieldValue('excludedWorkloadTypes', toggleItemInArray(excludedWorkloadTypes, type));
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
        if (isWorkloadTypeExclusionDisabled && excludedWorkloadTypes.length > 0) {
            setFieldValue('excludedWorkloadTypes', []);
        }
    }, [isWorkloadTypeExclusionDisabled, excludedWorkloadTypes.length, setFieldValue]);

    // @TODO: Consider using a custom component for the multi-select typeahead dropdown. PolicyCategoriesSelectField.tsx is a good example too.
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
            <FlexItem flex={{ default: 'flex_1' }} className="pf-v6-u-p-lg">
                <Title headingLevel="h2">Resources</Title>
                <div className="pf-v6-u-mt-sm">
                    Configure the resources to be applied to, or excluded from this policy.
                </div>
            </FlexItem>
            <Divider component="div" />
            {isAllScopingDisabled && (
                <Alert
                    className="pf-v6-u-mt-lg pf-v6-u-mx-lg"
                    isInline
                    variant="info"
                    title="The selected event source does not support resource targeting."
                    component="p"
                />
            )}
            <Flex direction={{ default: 'column' }} className="pf-v6-u-p-lg">
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Flex direction={{ default: 'column' }}>
                            <Title headingLevel="h3">Included resources</Title>
                            <div>
                                Define one or more clusters, namespaces of workloads (if applicable)
                                to apply this policy to. If no inclusions are configured, the policy
                                will apply to all resources in your environment, except those
                                excluded.
                            </div>
                            <PolicyScopeRE2Description />
                        </Flex>
                    </FlexItem>
                    <FlexItem className="pf-v6-u-pr-md" alignSelf={{ default: 'alignSelfCenter' }}>
                        <Button
                            variant="secondary"
                            onClick={addNewInclusionScope}
                            isDisabled={isAllScopingDisabled}
                        >
                            Add inclusion
                        </Button>
                    </FlexItem>
                </Flex>
                <FlexItem>
                    <Grid hasGutter md={6} xl2={4}>
                        {scope?.map((_, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index}>
                                {isFeatureFlagEnabled('ROX_LABEL_BASED_POLICY_SCOPING') ? (
                                    <InclusionScopeCard
                                        index={index}
                                        scope={scope[index]}
                                        clusters={clusters}
                                        handleChange={handleChange}
                                        setFieldValue={setFieldValue}
                                        onDelete={() => deleteInclusionScope(index)}
                                    />
                                ) : (
                                    <PolicyScopeCardLegacy
                                        type="inclusion"
                                        name={`scope[${index}]`}
                                        clusters={clusters}
                                        onDelete={() => deleteInclusionScope(index)}
                                        hasAuditLogEventSource={hasAuditLogEventSource}
                                    />
                                )}
                            </GridItem>
                        ))}
                    </Grid>
                </FlexItem>
            </Flex>
            <Divider component="div" />
            <Flex direction={{ default: 'column' }} className="pf-v6-u-p-lg">
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Flex direction={{ default: 'column' }}>
                            <Title headingLevel="h3">Excluded resources</Title>
                            <div>
                                {showWorkloadTypeExclusion
                                    ? 'Exclude resources from this policy by workload type, by scope, or both.'
                                    : 'Define one or more clusters, namespaces or workloads (if applicable) to be excluded from this policy.'}
                            </div>
                            {!showWorkloadTypeExclusion && <PolicyScopeRE2Description />}
                        </Flex>
                    </FlexItem>
                    {!showWorkloadTypeExclusion && (
                        <FlexItem
                            className="pf-v6-u-pr-md"
                            alignSelf={{ default: 'alignSelfCenter' }}
                        >
                            <Button
                                variant="secondary"
                                isDisabled={!hasDeployOrRuntimeLifecycle || isAllScopingDisabled}
                                onClick={addNewExclusionDeploymentScope}
                            >
                                Add exclusion
                            </Button>
                        </FlexItem>
                    )}
                </Flex>
                {showWorkloadTypeExclusion && (
                    <>
                        <FlexItem>
                            <Stack hasGutter>
                                <Title headingLevel="h4">By workload type</Title>
                                <div>Exclude workload types when evaluating this policy.</div>
                                <FormGroup fieldId="workload-type-exclusion" role="group">
                                    <Stack hasGutter>
                                        {policyWorkloadTypes.map((type) => (
                                            <Checkbox
                                                key={type}
                                                id={`exclude-workload-type-${type}`}
                                                label={policyWorkloadTypeLabels[type]}
                                                isChecked={excludedWorkloadTypes.includes(type)}
                                                isDisabled={isWorkloadTypeExclusionDisabled}
                                                onChange={() => handleChangeWorkloadType(type)}
                                            />
                                        ))}
                                        <Alert
                                            isInline
                                            isPlain
                                            variant="info"
                                            title={getWorkloadTypeExclusionInfo()}
                                            component="p"
                                        />
                                    </Stack>
                                </FormGroup>
                            </Stack>
                        </FlexItem>
                        <Divider component="div" />
                        <Flex>
                            <FlexItem flex={{ default: 'flex_1' }}>
                                <Flex direction={{ default: 'column' }}>
                                    <Title headingLevel="h4">By scope</Title>
                                    <div>
                                        Define one or more clusters, namespaces or workloads (if
                                        applicable) to be excluded from this policy.
                                    </div>
                                    <PolicyScopeRE2Description />
                                </Flex>
                            </FlexItem>
                            <FlexItem
                                className="pf-v6-u-pr-md"
                                alignSelf={{ default: 'alignSelfCenter' }}
                            >
                                <Button
                                    variant="secondary"
                                    isDisabled={
                                        !hasDeployOrRuntimeLifecycle || isAllScopingDisabled
                                    }
                                    onClick={addNewExclusionDeploymentScope}
                                >
                                    Add exclusion
                                </Button>
                            </FlexItem>
                        </Flex>
                    </>
                )}
                <FlexItem>
                    <Grid hasGutter md={6} xl={4}>
                        {excludedDeploymentScopes?.map((_, index) => (
                            // eslint-disable-next-line react/no-array-index-key
                            <GridItem key={index}>
                                {isFeatureFlagEnabled('ROX_LABEL_BASED_POLICY_SCOPING') ? (
                                    <ExclusionScopeCard
                                        index={index}
                                        excludedDeploymentScope={excludedDeploymentScopes[index]}
                                        clusters={clusters}
                                        handleChange={handleChange}
                                        setFieldValue={setFieldValue}
                                        onDelete={() => deleteExclusionDeploymentScope(index)}
                                    />
                                ) : (
                                    <PolicyScopeCardLegacy
                                        type="exclusion"
                                        name={`excludedDeploymentScopes[${index}]`}
                                        clusters={clusters}
                                        onDelete={() => deleteExclusionDeploymentScope(index)}
                                        hasAuditLogEventSource={hasAuditLogEventSource}
                                    />
                                )}
                            </GridItem>
                        ))}
                    </Grid>
                </FlexItem>
            </Flex>
            <Divider component="div" />
            <Flex direction={{ default: 'column' }} className="pf-v6-u-p-lg">
                <FlexItem flex={{ default: 'flex_1' }}>
                    <Title headingLevel="h3">Exclude images</Title>
                    <div className="pf-v6-u-mt-sm">
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
                                    onClick={() => setIsExcludeImagesOpen((prev) => !prev)}
                                    innerRef={toggleRef}
                                    isExpanded={isExcludeImagesOpen}
                                    isDisabled={
                                        hasAuditLogEventSource ||
                                        !hasBuildLifecycle ||
                                        isAllScopingDisabled
                                    }
                                    className="pf-v6-u-w-100"
                                >
                                    <TextInputGroup isPlain>
                                        <TextInputGroupMain
                                            value={filterValue}
                                            onClick={() => setIsExcludeImagesOpen((prev) => !prev)}
                                            onChange={(_event, value) => setFilterValue(value)}
                                            autoComplete="off"
                                            placeholder="Select images to exclude"
                                        >
                                            {excludedImageNames.length > 0 && (
                                                <LabelGroup>
                                                    {excludedImageNames.map((image) => (
                                                        <Label
                                                            variant="outline"
                                                            key={image}
                                                            onClose={(event) => {
                                                                event.stopPropagation();
                                                                handleChangeMultiSelect(
                                                                    event,
                                                                    image
                                                                );
                                                            }}
                                                        >
                                                            {image}
                                                        </Label>
                                                    ))}
                                                </LabelGroup>
                                            )}
                                        </TextInputGroupMain>
                                        <TextInputGroupUtilities>
                                            {excludedImageNames.length > 0 && (
                                                <Button
                                                    icon={<TimesIcon />}
                                                    variant="plain"
                                                    onClick={(event) => {
                                                        event.stopPropagation();
                                                        setFieldValue('excludedImageNames', []);
                                                        setFilterValue('');
                                                    }}
                                                    aria-label="Clear input value"
                                                />
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
