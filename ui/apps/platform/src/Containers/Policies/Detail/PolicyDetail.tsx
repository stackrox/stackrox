import React, { ReactElement, useState } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    AlertGroup,
    Breadcrumb,
    BreadcrumbItem,
    Label,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    Divider,
    PageSection,
} from '@patternfly/react-core';
import {
    Dropdown,
    DropdownItem,
    DropdownSeparator,
    DropdownToggle,
} from '@patternfly/react-core/deprecated';
import { CaretDownIcon } from '@patternfly/react-icons';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import useToasts, { Toast } from 'hooks/patternfly/useToasts';
import { policiesBasePath } from 'routePaths';
import { deletePolicy, exportPolicies } from 'services/PoliciesService';
import { ClientPolicy } from 'types/policy.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import PolicyDetailContent from './PolicyDetailContent';

function formatUpdateDisabledStateAction(disabled: boolean) {
    return disabled ? 'Enable policy' : 'Disable policy';
}

type PolicyDetailProps = {
    handleUpdateDisabledState: (id: string, disabled: boolean) => Promise<void>;
    hasWriteAccessForPolicy: boolean;
    policy: ClientPolicy;
};

function PolicyDetail({
    handleUpdateDisabledState,
    hasWriteAccessForPolicy,
    policy,
}: PolicyDetailProps): ReactElement {
    const history = useHistory();

    const [isRequesting, setIsRequesting] = useState(false);
    const [requestError, setRequestError] = useState<ReactElement | null>(null);
    const [isActionsOpen, setIsActionsOpen] = useState(false);
    const [isDeleteOpen, setIsDeleteOpen] = useState(false);

    const { toasts, addToast, removeToast } = useToasts();

    const { disabled, id, isDefault, name } = policy;

    function onSelectActions() {
        setIsActionsOpen(false);
    }

    function onToggleActions(isOpen) {
        setIsActionsOpen(isOpen);
    }

    function onEditPolicy() {
        history.push({
            pathname: `${policiesBasePath}/${id}`,
            search: 'action=edit',
        });
    }

    function onClonePolicy() {
        history.push({
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
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                addToast('Could not export the policy', 'danger', message);
            })
            .finally(() => {
                setIsRequesting(false);
            });
    }

    function onUpdateDisabledState() {
        setRequestError(null);
        setIsRequesting(true);
        handleUpdateDisabledState(id, !disabled)
            // If success, callback function updates policy prop.
            .catch((error) => {
                setRequestError(
                    <Alert
                        title={`Request failed: ${formatUpdateDisabledStateAction(disabled)}`}
                        component="p"
                        variant="danger"
                        isInline
                        actionClose={
                            <AlertActionCloseButton onClose={() => setRequestError(null)} />
                        }
                    >
                        {getAxiosErrorMessage(error)}
                    </Alert>
                );
            })
            .finally(() => {
                setIsRequesting(false);
            });
    }

    function onConfirmDeletePolicy() {
        setRequestError(null);
        setIsRequesting(true);
        deletePolicy(id)
            .then(() => {
                // Route change causes policy table page to request policies.
                history.goBack();
            })
            .catch((error) => {
                setRequestError(
                    <Alert
                        title="Request failed: Delete policy"
                        component="p"
                        variant="danger"
                        isInline
                        actionClose={
                            <AlertActionCloseButton onClose={() => setRequestError(null)} />
                        }
                    >
                        {getAxiosErrorMessage(error)}
                    </Alert>
                );
            })
            .finally(() => {
                setIsRequesting(false);
                setIsDeleteOpen(false);
            });
    }

    function onCancelDeletePolicy() {
        setIsDeleteOpen(false);
    }

    return (
        <>
            <PageSection variant="light" isFilled id="policy-page" className="pf-v5-u-pb-0">
                <Breadcrumb className="pf-v5-u-mb-md">
                    <BreadcrumbItemLink to={policiesBasePath}>Policies</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{name}</BreadcrumbItem>
                </Breadcrumb>
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
                        <ToolbarItem align={{ default: 'alignRight' }}>
                            <Dropdown
                                onSelect={onSelectActions}
                                position="right"
                                toggle={
                                    <DropdownToggle
                                        isDisabled={isRequesting}
                                        toggleVariant="primary"
                                        onToggle={(_event, isOpen) => onToggleActions(isOpen)}
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
                                                  {formatUpdateDisabledStateAction(disabled)}
                                              </DropdownItem>,
                                              <DropdownSeparator key="Separator" />,
                                              <DropdownItem
                                                  key="Delete policy"
                                                  component="button"
                                                  isDisabled={isDefault}
                                                  onClick={() => setIsDeleteOpen(true)}
                                              >
                                                  {isDefault
                                                      ? 'Cannot delete a default policy'
                                                      : 'Delete policy'}
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
            </PageSection>
            <PageSection variant="light" isFilled className="pf-v5-u-pt-0">
                {requestError}
                <Title headingLevel="h2" className="pf-v5-u-mb-md">
                    Policy details
                </Title>
                <Divider component="div" className="pf-v5-u-pb-md" />
                <PolicyDetailContent policy={policy} />
                <AlertGroup isToast isLiveRegion>
                    {toasts.map(({ key, variant, title, children }: Toast) => (
                        <Alert
                            variant={variant}
                            title={title}
                            component="p"
                            timeout={4000}
                            onTimeout={() => removeToast(key)}
                            actionClose={
                                <AlertActionCloseButton
                                    title={title}
                                    variantLabel={`${variant} alert`}
                                    onClose={() => removeToast(key)}
                                />
                            }
                            key={key}
                        >
                            {children}
                        </Alert>
                    ))}
                </AlertGroup>
            </PageSection>
            <ConfirmationModal
                ariaLabel="Confirm delete"
                confirmText="Delete"
                isLoading={isRequesting}
                isOpen={isDeleteOpen}
                onConfirm={onConfirmDeletePolicy}
                onCancel={onCancelDeletePolicy}
            >
                Are you sure you want to delete this policy?
            </ConfirmationModal>
        </>
    );
}

export default PolicyDetail;
