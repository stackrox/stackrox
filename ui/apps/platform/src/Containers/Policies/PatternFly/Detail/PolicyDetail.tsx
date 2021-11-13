import React, { ReactElement, useState } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    AlertGroup,
    AlertVariant,
    Dropdown,
    DropdownItem,
    DropdownSeparator,
    DropdownToggle,
    Label,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { CaretDownIcon } from '@patternfly/react-icons';

import useToasts from 'hooks/useToasts';
import { policiesBasePathPatternFly as policiesBasePath } from 'routePaths';
import { exportPolicies } from 'services/PoliciesService';
import { Policy } from 'types/policy.proto';

type PolicyDetailProps = {
    hasWriteAccessForPolicy: boolean;
    policy: Policy;
};

function PolicyDetail({ hasWriteAccessForPolicy, policy }: PolicyDetailProps): ReactElement {
    const history = useHistory();

    const [isRequesting, setIsRequesting] = useState(false);
    const [isActionsOpen, setIsActionsOpen] = useState(false);

    const { toasts, addToast, removeToast } = useToasts();

    const { disabled, id, name } = policy;

    function onSelectActions() {
        setIsActionsOpen(false);
    }

    function onToggleActions(isOpen) {
        setIsActionsOpen(isOpen);
    }

    function onEditPolicy() {
        history.replace({
            pathname: `${policiesBasePath}/${id}`,
            search: 'action=edit',
        });
    }

    function onClonePolicy() {
        history.replace({
            pathname: `${policiesBasePath}/${id}`,
            search: 'action=clone',
        });
    }

    function onExportPolicy() {
        setIsRequesting(true);
        exportPolicies([id])
            .then(() => {
                addToast('Successfully exported policy', 'success');
            })
            .catch(({ response }) => {
                addToast('Could not export policy', 'danger', response.data.message);
            })
            .finally(() => {
                setIsRequesting(false);
            });
    }

    function onUpdateDisabledState() {
        // TODO handleUpdateDisabledState(id, !disabled) callback?
    }

    function onDeletePolicy() {
        // TODO handleDeletePolicy(id) callback?
    }

    return (
        <>
            <Toolbar inset={{ default: 'insetNone' }}>
                <ToolbarContent>
                    <ToolbarItem>
                        <Title headingLevel="h1">{name}</Title>
                    </ToolbarItem>
                    <ToolbarItem>
                        {disabled ? (
                            <Label color="grey">Disabled</Label>
                        ) : (
                            <Label color="green">Enabled</Label>
                        )}
                    </ToolbarItem>
                    <ToolbarItem alignment={{ default: 'alignRight' }}>
                        <Dropdown
                            onSelect={onSelectActions}
                            position="right"
                            toggle={
                                <DropdownToggle
                                    isDisabled={isRequesting}
                                    isPrimary
                                    onToggle={onToggleActions}
                                    toggleIndicator={CaretDownIcon}
                                >
                                    Actions
                                </DropdownToggle>
                            }
                            isOpen={isActionsOpen}
                            dropdownItems={
                                hasWriteAccessForPolicy
                                    ? [
                                          <DropdownItem
                                              key="Edit policy"
                                              component="button"
                                              onClick={onEditPolicy}
                                          >
                                              Edit policy
                                          </DropdownItem>,
                                          <DropdownItem
                                              key="Clone policy"
                                              component="button"
                                              onClick={onClonePolicy}
                                          >
                                              Clone policy
                                          </DropdownItem>,
                                          <DropdownItem
                                              key="Export policy to JSON"
                                              component="button"
                                              onClick={onExportPolicy}
                                          >
                                              Export policy to JSON
                                          </DropdownItem>,
                                          <DropdownItem
                                              key="Enable/Disable policy"
                                              component="button"
                                              onClick={onUpdateDisabledState}
                                          >
                                              {disabled ? 'Enable policy' : 'Disable policy'}
                                          </DropdownItem>,
                                          <DropdownSeparator key="Separator" />,
                                          <DropdownItem
                                              key="Delete policy"
                                              component="button"
                                              onClick={onDeletePolicy}
                                          >
                                              Delete policy
                                          </DropdownItem>,
                                      ]
                                    : [
                                          <DropdownItem
                                              key="Export policy to JSON"
                                              component="button"
                                              onClick={onExportPolicy}
                                          >
                                              Export policy to JSON
                                          </DropdownItem>,
                                      ]
                            }
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Title headingLevel="h2">Policy overview</Title>
            TODO
            <Title headingLevel="h2">MITRE ATT&amp;CK</Title>
            TODO
            <Title headingLevel="h2">Policy criteria</Title>
            TODO
            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }) => (
                    <Alert
                        variant={AlertVariant[variant]}
                        title={title}
                        timeout={4000}
                        actionClose={
                            <AlertActionCloseButton
                                title={title}
                                variantLabel={`${variant as string} alert`}
                                onClose={() => removeToast(key)}
                            />
                        }
                        key={key}
                    >
                        {children}
                    </Alert>
                ))}
            </AlertGroup>
        </>
    );
}

export default PolicyDetail;
