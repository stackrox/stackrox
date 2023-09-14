import React, { ReactElement, useEffect, useState } from 'react';
import { FormikContextType } from 'formik';
import {
    Alert,
    AlertVariant,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    TextInput,
    Tooltip,
} from '@patternfly/react-core';
import { OutlinedQuestionCircleIcon } from '@patternfly/react-icons';

import {
    AccessScope,
    EffectiveAccessScopeCluster,
    LabelSelector,
    LabelSelectorsKey,
    computeEffectiveAccessScopeClusters,
    getIsUnrestrictedAccessScopeId,
} from 'services/AccessScopesService';

import {
    LabelSelectorsEditingState,
    getIsValidRules,
    getTemporarilyValidRules,
} from './accessScopes.utils';
import EffectiveAccessScopeTable from './EffectiveAccessScopeTable';
import LabelInclusion from './LabelInclusion';

const labelIconEffectiveAccessScope = (
    <Tooltip
        content="A list of the allowed clusters and namespaces in each cluster that a user role can access. Resources may be included manually or through label selector rules."
        isContentLeftAligned
        maxWidth="24em"
    >
        <div className="pf-c-button pf-m-plain pf-m-small">
            <OutlinedQuestionCircleIcon />
        </div>
    </Tooltip>
);

const labelIconLabelInclusion = (
    <Tooltip
        content={
            <div>
                Configure label selection rules to provide access to clusters and namespaces based
                on their labels
            </div>
        }
        isContentLeftAligned
        maxWidth="24em"
    >
        <div className="pf-c-button pf-m-plain pf-m-small">
            <OutlinedQuestionCircleIcon />
        </div>
    </Tooltip>
);

export type AccessScopeFormProps = {
    hasAction: boolean;
    alertSubmit: ReactElement | null;
    formik: FormikContextType<AccessScope>;
};

function AccessScopeForm({ hasAction, alertSubmit, formik }: AccessScopeFormProps): ReactElement {
    const [counterComputing, setCounterComputing] = useState(0);
    const [alertCompute, setAlertCompute] = useState<ReactElement | null>(null);
    const [clusters, setClusters] = useState<EffectiveAccessScopeCluster[]>([]);

    // Disable Save button while editing label selectors.
    const [labelSelectorsEditingState, setLabelSelectorsEditingState] =
        useState<LabelSelectorsEditingState>({
            clusterLabelSelectors: -1,
            namespaceLabelSelectors: -1,
        });

    const { errors, handleChange, setFieldValue, values } = formik;

    /*
     * A label selector or set requirement is temporarily invalid when it is added,
     * before its first requirement or value has been added.
     */
    const isValidRules =
        !getIsUnrestrictedAccessScopeId(values.id) && getIsValidRules(values.rules);
    useEffect(() => {
        if (getIsUnrestrictedAccessScopeId(values.id)) {
            return;
        }
        setCounterComputing((counterPrev) => counterPrev + 1);
        computeEffectiveAccessScopeClusters(
            isValidRules ? values.rules : getTemporarilyValidRules(values.rules)
        )
            .then((clustersComputed) => {
                setClusters(clustersComputed);
                setAlertCompute(null);
            })
            .catch((error) => {
                setAlertCompute(
                    <Alert
                        title="Compute effective access scope failed"
                        variant={AlertVariant.danger}
                        isInline
                    >
                        {error.message}
                    </Alert>
                );
            })
            .finally(() => {
                setCounterComputing((counterPrev) => counterPrev - 1);
            });
    }, [isValidRules, values.rules]);

    function onChange(_value, event) {
        handleChange(event);
    }

    function handleIncludedClustersChange(clusterName: string, isChecked: boolean) {
        const { includedClusters } = values.rules;
        return setFieldValue(
            'rules.includedClusters',
            isChecked
                ? [...includedClusters, clusterName]
                : includedClusters.filter(
                      (includedClusterName) => includedClusterName !== clusterName
                  )
        );
    }

    function handleIncludedNamespacesChange(
        clusterName: string,
        namespaceName: string,
        isChecked: boolean
    ) {
        const { includedNamespaces } = values.rules;
        return setFieldValue(
            'rules.includedNamespaces',
            isChecked
                ? [...includedNamespaces, { clusterName, namespaceName }]
                : includedNamespaces.filter(
                      ({
                          clusterName: includedClusterName,
                          namespaceName: includedNamespaceName,
                      }) =>
                          includedClusterName !== clusterName ||
                          includedNamespaceName !== namespaceName
                  )
        );
    }

    function handleLabelSelectorsChange(
        labelSelectorsKey: LabelSelectorsKey,
        labelSelectors: LabelSelector[]
    ) {
        return setFieldValue(`rules.${labelSelectorsKey}`, labelSelectors);
    }

    const isViewing = !hasAction;

    const nameErrorMessage = values.name.length !== 0 && errors.name ? errors.name : '';
    const nameValidatedState = nameErrorMessage ? 'error' : 'default';

    return (
        <Form id="access-scope-form">
            {alertSubmit}
            <FormGroup
                label="Name"
                fieldId="name"
                isRequired
                validated={nameValidatedState}
                helperTextInvalid={nameErrorMessage}
                className="pf-m-horizontal"
            >
                <TextInput
                    type="text"
                    id="name"
                    value={values.name}
                    validated={nameValidatedState}
                    onChange={onChange}
                    isDisabled={isViewing}
                    isRequired
                    className="pf-m-limit-width"
                />
            </FormGroup>
            <FormGroup label="Description" fieldId="description" className="pf-m-horizontal">
                <TextInput
                    type="text"
                    id="description"
                    value={values.description}
                    onChange={onChange}
                    isDisabled={isViewing}
                />
            </FormGroup>
            {alertCompute}
            {!getIsUnrestrictedAccessScopeId(values.id) && (
                <Flex
                    direction={{ default: 'row' }}
                    spaceItems={{ default: 'spaceItemsSm', xl: 'spaceItemsLg' }}
                >
                    <FlexItem className="pf-u-flex-basis-0" flex={{ default: 'flex_1' }}>
                        <FormGroup
                            label="Allowed resources"
                            fieldId="effectiveAccessScope"
                            labelIcon={labelIconEffectiveAccessScope}
                        >
                            <EffectiveAccessScopeTable
                                counterComputing={counterComputing}
                                clusters={clusters}
                                includedClusters={values.rules.includedClusters}
                                includedNamespaces={values.rules.includedNamespaces}
                                handleIncludedClustersChange={handleIncludedClustersChange}
                                handleIncludedNamespacesChange={handleIncludedNamespacesChange}
                                hasAction={hasAction}
                            />
                        </FormGroup>
                    </FlexItem>
                    <FlexItem className="pf-u-flex-basis-0" flex={{ default: 'flex_1' }}>
                        <FormGroup
                            label="Label selection rules"
                            fieldId="labelInclusion"
                            labelIcon={labelIconLabelInclusion}
                        >
                            <LabelInclusion
                                clusterLabelSelectors={values.rules.clusterLabelSelectors}
                                namespaceLabelSelectors={values.rules.namespaceLabelSelectors}
                                hasAction={hasAction}
                                labelSelectorsEditingState={labelSelectorsEditingState}
                                setLabelSelectorsEditingState={setLabelSelectorsEditingState}
                                handleLabelSelectorsChange={handleLabelSelectorsChange}
                            />
                        </FormGroup>
                    </FlexItem>
                </Flex>
            )}
        </Form>
    );
}

export default AccessScopeForm;
