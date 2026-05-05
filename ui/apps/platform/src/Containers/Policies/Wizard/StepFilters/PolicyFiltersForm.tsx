import { useEffect } from 'react';
import {
    Alert,
    Checkbox,
    Content,
    Divider,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    Radio,
    Stack,
    Title,
} from '@patternfly/react-core';
import { useFormikContext } from 'formik';

import type { ClientPolicy, SkipImageLayers } from 'types/policy.proto';
import { policyCriteriaCategories } from 'messages/common';
import { policyCriteriaDescriptors } from '../Step3/policyCriteriaDescriptors';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { toggleItemInArray } from 'utils/arrayUtils';

const imageCriteriaCategories = new Set<string>([
    policyCriteriaCategories.IMAGE_CONTENTS,
    policyCriteriaCategories.IMAGE_SCANNING,
]);

const imageRelatedFieldNames = new Set(
    policyCriteriaDescriptors
        .filter((d) => imageCriteriaCategories.has(d.category))
        .map((d) => d.name)
);

function hasImageCriteria(policy: ClientPolicy): boolean {
    return policy.policySections.some((section) =>
        section.policyGroups.some((group) => imageRelatedFieldNames.has(group.fieldName))
    );
}

const imageLayerDescriptions: Record<SkipImageLayers, string> = {
    SKIP_NONE: 'Policy will evaluate all image layers, including both base and application layers.',
    SKIP_BASE:
        'Base image layers (e.g., OS packages, system libraries) will be skipped. Only application layers added on top of the base image will be evaluated.',
    SKIP_APP:
        'Application layers added during the build will be skipped. Only base image layers will be evaluated.',
};

function PolicyFiltersForm() {
    const { values, setFieldValue } = useFormikContext<ClientPolicy>();
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const showContainerFilters = isFeatureFlagEnabled('ROX_INIT_CONTAINER_SUPPORT');
    const showImageLayerFilters = isFeatureFlagEnabled('ROX_POLICY_FILTERS_UI');

    const skipContainerTypes = values.evaluationFilter?.skipContainerTypes ?? [];
    const skipImageLayers = values.evaluationFilter?.skipImageLayers ?? 'SKIP_NONE';

    const hasDeployOrRuntime =
        values.lifecycleStages.includes('DEPLOY') || values.lifecycleStages.includes('RUNTIME');
    const isBuildOnly =
        values.lifecycleStages.length === 1 && values.lifecycleStages.includes('BUILD');

    const containerFilterDisabled = !hasDeployOrRuntime;
    const imageLayerFilterDisabled = !hasImageCriteria(values);

    useEffect(() => {
        if (imageLayerFilterDisabled && skipImageLayers !== 'SKIP_NONE') {
            setFieldValue('evaluationFilter.skipImageLayers', 'SKIP_NONE');
        }
    }, [imageLayerFilterDisabled, skipImageLayers, setFieldValue]);

    useEffect(() => {
        if (containerFilterDisabled && skipContainerTypes.length > 0) {
            setFieldValue('evaluationFilter.skipContainerTypes', []);
        }
    }, [containerFilterDisabled, skipContainerTypes.length, setFieldValue]);

    const skipInit = skipContainerTypes.includes('SKIP_INIT');

    const containerTypeDescription = skipInit
        ? 'Policy will skip init containers.'
        : 'Policy will evaluate all container types.';

    function handleSkipInitChange() {
        setFieldValue(
            'evaluationFilter.skipContainerTypes',
            toggleItemInArray(skipContainerTypes, 'SKIP_INIT')
        );
    }

    function handleImageLayerChange(layer: SkipImageLayers) {
        setFieldValue('evaluationFilter.skipImageLayers', layer);
    }

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
            <FlexItem flex={{ default: 'flex_1' }} className="pf-v6-u-p-lg">
                <Stack hasGutter>
                    <Title headingLevel="h2">Filters</Title>
                    <Content component="p">
                        Control which parts of an image or workload this policy evaluates.
                    </Content>
                </Stack>
            </FlexItem>
            <Divider component="div" />

            {showImageLayerFilters && (
                <>
                    <FlexItem className="pf-v6-u-p-lg">
                        <Stack hasGutter>
                            <Title headingLevel="h3">Image layers</Title>
                            <Content component="p">
                                Select which image layers this policy evaluates for components,
                                CVEs, and Dockerfile instructions.
                            </Content>
                            {imageLayerFilterDisabled && (
                                <Alert
                                    isInline
                                    variant="info"
                                    title="Image layer filtering requires at least one criterion from Image Contents or Image Scanning."
                                    component="p"
                                />
                            )}
                            {!imageLayerFilterDisabled && isBuildOnly && (
                                <Alert
                                    isInline
                                    variant="info"
                                    title="Image layer filtering applies to all lifecycle stages, including Build."
                                    component="p"
                                />
                            )}
                            <Form>
                                <FormGroup fieldId="image-layer-filter" role="radiogroup">
                                    <Stack hasGutter>
                                        <Flex
                                            direction={{ default: 'column' }}
                                            spaceItems={{ default: 'spaceItemsXs' }}
                                        >
                                            <Radio
                                                label="Evaluate all layers"
                                                id="image-layer-all"
                                                name="imageLayerFilter"
                                                isChecked={skipImageLayers === 'SKIP_NONE'}
                                                isDisabled={imageLayerFilterDisabled}
                                                onChange={() => handleImageLayerChange('SKIP_NONE')}
                                            />
                                            <Radio
                                                label="Skip base image layers"
                                                id="image-layer-base"
                                                name="imageLayerFilter"
                                                isChecked={skipImageLayers === 'SKIP_BASE'}
                                                isDisabled={imageLayerFilterDisabled}
                                                onChange={() => handleImageLayerChange('SKIP_BASE')}
                                            />
                                            <Radio
                                                label="Skip application layers"
                                                id="image-layer-app"
                                                name="imageLayerFilter"
                                                isChecked={skipImageLayers === 'SKIP_APP'}
                                                isDisabled={imageLayerFilterDisabled}
                                                onChange={() => handleImageLayerChange('SKIP_APP')}
                                            />
                                        </Flex>
                                        <Alert
                                            isInline
                                            isPlain
                                            variant="info"
                                            title={
                                                imageLayerFilterDisabled
                                                    ? 'Add criteria from Image Contents or Image Scanning in the Rules step to enable image layer filtering.'
                                                    : imageLayerDescriptions[skipImageLayers]
                                            }
                                            component="p"
                                        />
                                    </Stack>
                                </FormGroup>
                            </Form>
                        </Stack>
                    </FlexItem>
                    <Divider component="div" />
                </>
            )}

            {showContainerFilters && (
                <FlexItem className="pf-v6-u-p-lg">
                    <Stack hasGutter>
                        <Title headingLevel="h3">Container types</Title>
                        <Content component="p">
                            Select which container types to skip when evaluating this policy.
                        </Content>
                        {containerFilterDisabled && (
                            <Alert
                                isInline
                                variant="info"
                                title="Container type filters require the Deploy or Runtime lifecycle stage."
                                component="p"
                            />
                        )}
                        <Form>
                            <FormGroup fieldId="container-type-filter" role="group">
                                <Stack hasGutter>
                                    <Checkbox
                                        label="Skip init containers"
                                        id="skip-init-containers"
                                        isChecked={skipInit}
                                        isDisabled={containerFilterDisabled}
                                        onChange={handleSkipInitChange}
                                    />
                                    <Alert
                                        isInline
                                        isPlain
                                        variant="info"
                                        title={
                                            containerFilterDisabled
                                                ? 'Container type filtering is only available for policies with the Deploy or Runtime lifecycle stage.'
                                                : containerTypeDescription
                                        }
                                        component="p"
                                    />
                                </Stack>
                            </FormGroup>
                        </Form>
                    </Stack>
                </FlexItem>
            )}
        </Flex>
    );
}

export default PolicyFiltersForm;
