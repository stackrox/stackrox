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
    Stack,
    Title,
} from '@patternfly/react-core';
import { useFormikContext } from 'formik';

import type { ClientPolicy } from 'types/policy.proto';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { toggleItemInArray } from 'utils/arrayUtils';

function PolicyFiltersForm() {
    const { values, setFieldValue } = useFormikContext<ClientPolicy>();
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const showContainerFilters =
        isFeatureFlagEnabled('ROX_EVALUATION_FILTER') &&
        isFeatureFlagEnabled('ROX_INIT_CONTAINER_SUPPORT');

    const skipContainerTypes = values.evaluationFilter?.skipContainerTypes ?? [];

    const hasDeployOrRuntime =
        values.lifecycleStages.includes('DEPLOY') || values.lifecycleStages.includes('RUNTIME');

    const containerFilterDisabled = !hasDeployOrRuntime;

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
