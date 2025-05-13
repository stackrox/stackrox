import React from 'react';
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
} from '@patternfly/react-core';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';

import { ClientPolicy } from 'types/policy.proto';
import { ListImage } from 'types/image.proto';
import { ListDeployment } from 'types/deployment.proto';
import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import { getImages } from 'services/imageService';
import { fetchDeploymentsWithProcessInfoLegacy as fetchDeploymentsWithProcessInfo } from 'services/DeploymentsService';
import PolicyScopeCard from './PolicyScopeCard';

function PolicyScopeForm() {
    const [isExcludeImagesOpen, setIsExcludeImagesOpen] = React.useState(false);
    const [images, setImages] = React.useState<ListImage[]>([]);
    const [deployments, setDeployments] = React.useState<ListDeployment[]>([]);
    const { clusters } = useFetchClustersForPermissions(['Deployment']);
    const { values, setFieldValue } = useFormikContext<ClientPolicy>();
    const { scope, excludedDeploymentScopes, excludedImageNames } = values;

    const hasAuditLogEventSource = values.eventSource === 'AUDIT_LOG_EVENT';
    const hasBuildLifecycle = values.lifecycleStages.includes('BUILD');
    const hasDeployOrRuntimeLifecycle =
        values.lifecycleStages.includes('DEPLOY') || values.lifecycleStages.includes('RUNTIME');

    function addNewInclusionScope() {
        setFieldValue('scope', [...scope, {}]);
    }

    function deleteInclusionScope(index) {
        const newScope = scope.filter((_, i) => i !== index);
        setFieldValue('scope', newScope);
    }

    function addNewExclusionDeploymentScope() {
        setFieldValue('excludedDeploymentScopes', [...excludedDeploymentScopes, {}]);
    }

    function deleteExclusionDeploymentScope(index) {
        const newScope = excludedDeploymentScopes.filter((_, i) => i !== index);
        setFieldValue('excludedDeploymentScopes', newScope);
    }

    function handleChangeMultiSelect(e, selectedImage) {
        setIsExcludeImagesOpen(false);
        if (excludedImageNames.includes(selectedImage)) {
            const newExclusions = excludedImageNames.filter((image) => image !== selectedImage);
            setFieldValue('excludedImageNames', newExclusions);
        } else {
            setFieldValue('excludedImageNames', [...excludedImageNames, selectedImage]);
        }
    }

    React.useEffect(() => {
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
                            onToggle={() => setIsExcludeImagesOpen(!isExcludeImagesOpen)}
                            isOpen={isExcludeImagesOpen}
                            variant="typeaheadmulti"
                            selections={excludedImageNames}
                            onSelect={handleChangeMultiSelect}
                            isCreatable
                            createText="Images starting with "
                            onCreateOption={() => {}}
                            isDisabled={hasAuditLogEventSource || !hasBuildLifecycle}
                            onClear={() => setFieldValue('excludedImageNames', [])}
                            placeholderText="Select images to exclude"
                            maxHeight="300px"
                        >
                            {images?.map((image) => (
                                <SelectOption key={image.name} value={image.name}>
                                    {image.name}
                                </SelectOption>
                            ))}
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
