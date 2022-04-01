import React, { useState, useEffect, ReactElement } from 'react';
import { Button, ButtonVariant, Flex, FlexItem, SelectOption } from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import {
    fetchAccessScopes,
    AccessScope,
    getIsDefaultAccessScopeId,
} from 'services/AccessScopesService';
import ResourceScopeFormModal from './ResourceScopeFormModal';

type ResourceScopeSelectionProps = {
    scopeId: string;
    setFieldValue: (field: string, value: any, shouldValidate?: boolean | undefined) => void;
    allowCreate: boolean;
};

function ResourceScopeSelection({
    scopeId,
    setFieldValue,
    allowCreate,
}: ResourceScopeSelectionProps): ReactElement {
    const [resourceScopes, setResourceScopes] = useState<AccessScope[]>([]);
    const [lastAddedResourceScopeId, setLastAddedResourceScopeId] = useState('');
    const [isResourceScopeModalOpen, setIsResourceScopeModalOpen] = useState(false);

    useEffect(() => {
        fetchAccessScopes()
            .then((response) => {
                const resourceScopesList = response || [];
                const filteredScopes = resourceScopesList.filter(
                    (scope) => !getIsDefaultAccessScopeId(scope.id)
                );
                setResourceScopes(filteredScopes);

                if (lastAddedResourceScopeId) {
                    onScopeChange('scopeId', lastAddedResourceScopeId);
                }
            })
            .catch(() => {
                // TODO display message when there is a place for minor errors
            });
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [lastAddedResourceScopeId]);

    function onToggleResourceScopeModal() {
        setIsResourceScopeModalOpen((current) => !current);
    }

    function onScopeChange(_id, selection) {
        setFieldValue('scopeId', selection);
    }

    return (
        <>
            <Flex alignItems={{ default: 'alignItemsFlexEnd' }}>
                <FlexItem>
                    <FormLabelGroup
                        className="pf-u-mb-md"
                        isRequired
                        label="Configure resource scope"
                        fieldId="scopeId"
                        touched={{}}
                        errors={{}}
                    >
                        <SelectSingle
                            id="scopeId"
                            value={scopeId}
                            handleSelect={onScopeChange}
                            isDisabled={false}
                            placeholderText="Select a scope"
                        >
                            {resourceScopes.map(({ id, name, description }) => (
                                <SelectOption key={id} value={id} description={description}>
                                    {name}
                                </SelectOption>
                            ))}
                        </SelectSingle>
                    </FormLabelGroup>
                </FlexItem>
                {allowCreate && (
                    <FlexItem>
                        <Button
                            className="pf-u-mb-md"
                            variant={ButtonVariant.secondary}
                            onClick={onToggleResourceScopeModal}
                        >
                            Create resource scope
                        </Button>
                    </FlexItem>
                )}
            </Flex>
            <ResourceScopeFormModal
                isOpen={isResourceScopeModalOpen}
                updateResourceScopeList={setLastAddedResourceScopeId}
                onToggleResourceScopeModal={onToggleResourceScopeModal}
                resourceScopes={resourceScopes}
            />
        </>
    );
}

export default ResourceScopeSelection;
