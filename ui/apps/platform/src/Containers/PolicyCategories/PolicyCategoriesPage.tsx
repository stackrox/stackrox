import React, { useState, useEffect } from 'react';
import {
    PageSection,
    Bullseye,
    Spinner,
    Divider,
    Button,
    Flex,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    AlertGroup,
    Alert,
    AlertVariant,
    AlertActionCloseButton,
} from '@patternfly/react-core';

import usePermissions from 'hooks/usePermissions';
import useToasts, { Toast } from 'hooks/patternfly/useToasts';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { PolicyCategory } from 'types/policy.proto';
import { getPolicyCategories } from 'services/PolicyCategoriesService';
import PolicyManagementHeader from 'Containers/PolicyManagement/PolicyManagementHeader';
import PolicyCategoriesListSection from './PolicyCategoriesListSection';
import CreatePolicyCategoryModal from './CreatePolicyCategoryModal';

function PolicyCategoriesPage(): React.ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForPolicy = hasReadWriteAccess('WorkflowAdministration');

    const [isLoading, setIsLoading] = useState(false);
    const [errorMessage, setErrorMessage] = useState('');
    const [policyCategories, setPolicyCategories] = useState<PolicyCategory[]>([]);
    const { toasts, addToast, removeToast } = useToasts();
    const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
    const [selectedCategory, setSelectedCategory] = useState<PolicyCategory>();

    let listContent = (
        <PageSection variant="light" isFilled id="policies-table-loading">
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        </PageSection>
    );

    if (errorMessage) {
        listContent = (
            <PageSection variant="light" isFilled id="policies-table-error">
                <Bullseye>
                    <Alert variant="danger" title={errorMessage} component="div" />
                </Bullseye>
            </PageSection>
        );
    }

    if (!isLoading && !errorMessage) {
        listContent = (
            <PolicyCategoriesListSection
                policyCategories={policyCategories}
                addToast={addToast}
                setSelectedCategory={setSelectedCategory}
                selectedCategory={selectedCategory}
                refreshPolicyCategories={refreshPolicyCategories}
            />
        );
    }

    function refreshPolicyCategories() {
        getPolicyCategories()
            .then((categories) => {
                setPolicyCategories(categories);
                setErrorMessage('');
            })
            .catch((error) => {
                setPolicyCategories([]);
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => setIsLoading(false));
    }

    useEffect(() => {
        setIsLoading(true);
        refreshPolicyCategories();
    }, []);

    return (
        <>
            <PolicyManagementHeader currentTabTitle="Policy categories" />
            <Divider component="div" />
            <PageSection variant="light" className="pf-u-py-0">
                <Toolbar inset={{ default: 'insetNone' }}>
                    <ToolbarContent>
                        <ToolbarItem>
                            <div className="pf-u-font-size-sm">
                                Manage categories for your policies.
                            </div>
                        </ToolbarItem>
                        {hasWriteAccessForPolicy && (
                            <ToolbarItem alignment={{ default: 'alignRight' }}>
                                <Flex>
                                    <Button
                                        variant="primary"
                                        onClick={() => setIsCreateModalOpen(true)}
                                        isDisabled={isCreateModalOpen || !!selectedCategory}
                                    >
                                        Create category
                                    </Button>
                                </Flex>
                            </ToolbarItem>
                        )}
                    </ToolbarContent>
                </Toolbar>
            </PageSection>
            <Divider component="div" />
            {listContent}
            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }: Toast) => (
                    <Alert
                        variant={AlertVariant[variant]}
                        title={title}
                        component="div"
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
            <CreatePolicyCategoryModal
                isOpen={isCreateModalOpen}
                onClose={() => setIsCreateModalOpen(false)}
                refreshPolicyCategories={refreshPolicyCategories}
                addToast={addToast}
            />
        </>
    );
}

export default PolicyCategoriesPage;
