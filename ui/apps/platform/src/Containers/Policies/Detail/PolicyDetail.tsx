import React, { useState } from 'react';
import type { ReactElement } from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';
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
    Flex,
    FlexItem,
    DropdownItem,
} from '@patternfly/react-core';

import MenuDropdown from 'Components/PatternFly/MenuDropdown';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import useToasts from 'hooks/patternfly/useToasts';
import type { Toast } from 'hooks/patternfly/useToasts';
import { policiesBasePath } from 'routePaths';
import { deletePolicy, exportPolicies } from 'services/PoliciesService';
import { savePoliciesAsCustomResource } from 'services/PolicyCustomResourceService';
import type { ClientPolicy } from 'types/policy.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import PolicyDetailContent from './PolicyDetailContent';
import { isExternalPolicy } from '../policies.utils';

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
    const navigate = useNavigate();

    const [isRequesting, setIsRequesting] = useState(false);
    const [requestError, setRequestError] = useState<ReactElement | null>(null);
    const [isDeleteOpen, setIsDeleteOpen] = useState(false);
    const [isSaveAsCustomResourceOpen, setIsSaveAsCustomResourceOpen] = useState(false);

    const { toasts, addToast, removeToast } = useToasts();

    const { disabled, id, isDefault, name } = policy;

    function onEditPolicy() {
        navigate(`${policiesBasePath}/${id}?action=edit`);
    }

    function onClonePolicy() {
        navigate(`${policiesBasePath}/${id}?action=clone`);
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

    function onConfirmSavePolicyAsCustomResource() {
        setIsRequesting(true);
        savePoliciesAsCustomResource([id])
            .then(() => {
                addToast('Successfully saved policy as Custom Resource', 'success');
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                addToast('Could not save policy as Custom Resource', 'danger', message);
            })
            .finally(() => {
                setIsRequesting(false);
                setIsSaveAsCustomResourceOpen(false);
            });
    }

    function onCancelSavePolicyAsCustomResource() {
        setIsSaveAsCustomResourceOpen(false);
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
                navigate(-1);
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
                            <MenuDropdown
                                popperProps={{
                                    position: 'end',
                                }}
                                toggleText="Actions"
                                toggleVariant="primary"
                                isDisabled={isRequesting}
                            >
                                {hasWriteAccessForPolicy && (
                                    <DropdownItem key="Edit policy" onClick={onEditPolicy}>
                                        Edit policy
                                    </DropdownItem>
                                )}
                                {hasWriteAccessForPolicy && (
                                    <DropdownItem key="Clone policy" onClick={onClonePolicy}>
                                        Clone policy
                                    </DropdownItem>
                                )}
                                <DropdownItem key="Export policy to JSON" onClick={onExportPolicy}>
                                    Export policy to JSON
                                </DropdownItem>
                                <DropdownItem
                                    key="Save as Custom Resource"
                                    isDisabled={isDefault}
                                    description={
                                        isDefault
                                            ? 'Default policies cannot be saved as Custom Resource'
                                            : ''
                                    }
                                    onClick={() => setIsSaveAsCustomResourceOpen(true)}
                                >
                                    {isDefault
                                        ? 'Cannot save as Custom Resource'
                                        : 'Save as Custom Resource'}
                                </DropdownItem>
                                {hasWriteAccessForPolicy && (
                                    <DropdownItem
                                        key="Enable/Disable policy"
                                        onClick={onUpdateDisabledState}
                                    >
                                        {formatUpdateDisabledStateAction(disabled)}
                                    </DropdownItem>
                                )}
                                {hasWriteAccessForPolicy && (
                                    <Divider component="li" key="separator" />
                                )}
                                {hasWriteAccessForPolicy && (
                                    <DropdownItem
                                        key="Delete policy"
                                        isDisabled={isDefault}
                                        onClick={() => setIsDeleteOpen(true)}
                                    >
                                        {isDefault
                                            ? 'Cannot delete a default policy'
                                            : 'Delete policy'}
                                    </DropdownItem>
                                )}
                            </MenuDropdown>
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
                title={'Delete policy?'}
                ariaLabel="Confirm delete"
                confirmText="Delete"
                isLoading={isRequesting}
                isOpen={isDeleteOpen}
                onConfirm={onConfirmDeletePolicy}
                onCancel={onCancelDeletePolicy}
            >
                {isExternalPolicy(policy) ? (
                    <>
                        This policy is managed externally and will only be removed from the system
                        temporarily. The policy will not trigger violations until the next resync.
                    </>
                ) : (
                    <>
                        This policy will be permanently removed from the system and will no longer
                        trigger violations.
                    </>
                )}
            </ConfirmationModal>
            <ConfirmationModal
                title={`Save policy as Custom Resource?`}
                ariaLabel="Save as Custom Resource"
                confirmText="Yes"
                isLoading={isRequesting}
                isOpen={isSaveAsCustomResourceOpen}
                onConfirm={onConfirmSavePolicyAsCustomResource}
                onCancel={onCancelSavePolicyAsCustomResource}
                isDestructive={false}
            >
                <Flex>
                    <FlexItem>
                        Clicking <strong>Yes</strong> will save the policy as a Kubernetes custom
                        resource (YAML).
                    </FlexItem>
                    <FlexItem>
                        <strong>Important</strong>: If you are committing the saved custom resource
                        to a source control repository, replace the policy name in the{' '}
                        <code className="pf-v5-u-font-family-monospace">policyName</code> field to
                        avoid overwriting existing policies.
                    </FlexItem>
                </Flex>
            </ConfirmationModal>
        </>
    );
}

export default PolicyDetail;
