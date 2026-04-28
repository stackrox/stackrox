import {
    Alert,
    Checkbox,
    Divider,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    Radio,
    Title,
} from '@patternfly/react-core';
import { useFormikContext } from 'formik';

import type { ClientPolicy, SkipImageLayers } from 'types/policy.proto';
import { policyCriteriaCategories } from 'messages/common';
import { policyCriteriaDescriptors } from '../Step3/policyCriteriaDescriptors';
import useFeatureFlags from 'hooks/useFeatureFlags';

const imageCriteriaCategories = new Set([
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
    SKIP_NONE:
        'Policy will evaluate all image layers, including both base and application layers.',
    SKIP_BASE:
        'Policy will evaluate only application layers added on top of the base image. Base image layers (e.g., OS packages, system libraries) will be skipped.',
    SKIP_APP:
        'Policy will evaluate only base image layers. Application layers added during the build will be skipped.',
};

function PolicyFiltersForm() {
    const { values, setFieldValue } = useFormikContext<ClientPolicy>();
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const showContainerFilters = isFeatureFlagEnabled('ROX_INIT_CONTAINER_SUPPORT');
    const showImageLayerFilters = isFeatureFlagEnabled('ROX_IMAGE_LAYER_FILTER');

    const skipContainerTypes = values.evaluationFilter?.skipContainerTypes ?? [];
    const skipImageLayers = values.evaluationFilter?.skipImageLayers ?? 'SKIP_NONE';

    const hasDeployOrRuntime =
        values.lifecycleStages.includes('DEPLOY') || values.lifecycleStages.includes('RUNTIME');
    const isBuildOnly =
        values.lifecycleStages.length === 1 && values.lifecycleStages.includes('BUILD');

    const containerFilterDisabled = !hasDeployOrRuntime;
    const imageLayerFilterDisabled = !hasImageCriteria(values);

    const skipInit = skipContainerTypes.includes('SKIP_INIT');

    let containerTypeDescription = 'Policy will evaluate all container types.';
    if (skipInit) {
        containerTypeDescription = 'Policy will skip init containers.';
    }

    function handleSkipInitChange(_event: React.FormEvent, checked: boolean) {
        const updated = checked
            ? [...skipContainerTypes, 'SKIP_INIT' as const]
            : skipContainerTypes.filter((ct) => ct !== 'SKIP_INIT');
        setFieldValue('evaluationFilter.skipContainerTypes', updated);
    }

    function handleImageLayerChange(layer: SkipImageLayers) {
        setFieldValue('evaluationFilter.skipImageLayers', layer);
    }

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
            <FlexItem flex={{ default: 'flex_1' }} className="pf-v6-u-p-lg">
                <Title headingLevel="h2">Filters</Title>
                <div className="pf-v6-u-mt-sm">
                    Control which parts of an image or workload this policy evaluates.
                </div>
            </FlexItem>
            <Divider component="div" />

            {showImageLayerFilters && (
                <>
                    <Flex direction={{ default: 'column' }} className="pf-v6-u-p-lg">
                        <FlexItem>
                            <Title headingLevel="h3">Image layers</Title>
                            <div className="pf-v6-u-mt-sm pf-v6-u-mb-md">
                                Select which image layers this policy evaluates for components,
                                CVEs, and Dockerfile instructions.
                            </div>
                        </FlexItem>
                        {imageLayerFilterDisabled && (
                            <FlexItem>
                                <Alert
                                    isInline
                                    variant="info"
                                    title="Image layer filtering requires at least one criterion from Image Contents or Image Scanning."
                                    component="p"
                                />
                            </FlexItem>
                        )}
                        {!imageLayerFilterDisabled && isBuildOnly && (
                            <FlexItem>
                                <Alert
                                    isInline
                                    variant="info"
                                    title="Image layer filtering applies to all lifecycle stages, including Build."
                                    component="p"
                                />
                            </FlexItem>
                        )}
                        <FlexItem>
                            <Form>
                                <FormGroup fieldId="image-layer-filter" role="radiogroup">
                                    <Radio
                                        label="All layers"
                                        id="image-layer-all"
                                        name="imageLayerFilter"
                                        isChecked={skipImageLayers === 'SKIP_NONE'}
                                        isDisabled={imageLayerFilterDisabled}
                                        onChange={() => handleImageLayerChange('SKIP_NONE')}
                                    />
                                    <Radio
                                        label="Application layers only"
                                        id="image-layer-app"
                                        name="imageLayerFilter"
                                        isChecked={skipImageLayers === 'SKIP_BASE'}
                                        isDisabled={imageLayerFilterDisabled}
                                        onChange={() => handleImageLayerChange('SKIP_BASE')}
                                        className="pf-v6-u-mt-sm"
                                    />
                                    <Radio
                                        label="Base layers only"
                                        id="image-layer-base"
                                        name="imageLayerFilter"
                                        isChecked={skipImageLayers === 'SKIP_APP'}
                                        isDisabled={imageLayerFilterDisabled}
                                        onChange={() => handleImageLayerChange('SKIP_APP')}
                                        className="pf-v6-u-mt-sm"
                                    />
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
                                        className="pf-v6-u-mt-md"
                                    />
                                </FormGroup>
                            </Form>
                        </FlexItem>
                    </Flex>
                    <Divider component="div" />
                </>
            )}

            {showContainerFilters && (
                <Flex direction={{ default: 'column' }} className="pf-v6-u-p-lg">
                    <FlexItem>
                        <Title headingLevel="h3">Container types</Title>
                        <div className="pf-v6-u-mt-sm pf-v6-u-mb-md">
                            Select which container types to skip when evaluating this policy.
                        </div>
                    </FlexItem>
                    {containerFilterDisabled && (
                        <FlexItem>
                            <Alert
                                isInline
                                variant="info"
                                title="Container type filters require the Deploy or Runtime lifecycle stage."
                                component="p"
                            />
                        </FlexItem>
                    )}
                    <FlexItem>
                        <Form>
                            <FormGroup fieldId="container-type-filter" role="group">
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
                                    className="pf-v6-u-mt-md"
                                />
                            </FormGroup>
                        </Form>
                    </FlexItem>
                </Flex>
            )}
        </Flex>
    );
}

export default PolicyFiltersForm;
