import React, { useEffect, useState } from 'react';
import {
    Modal,
    ModalVariant,
    ModalBoxBody,
    ModalBoxFooter,
    Button,
    List,
    ListItem,
    ListComponent,
    OrderType,
    Flex,
    FlexItem,
    Panel,
    PanelMain,
    PanelMainBody,
} from '@patternfly/react-core';

import { getPolicies } from 'services/PoliciesService';
import { deletePolicyCategory } from 'services/PolicyCategoriesService';
import { PolicyCategory, ListPolicy } from 'types/policy.proto';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

type DeletePolicyCategoryModalType = {
    isOpen: boolean;
    onClose: () => void;
    addToast: (toast) => void;
    refreshPolicyCategories: () => void;
    selectedCategory?: PolicyCategory;
    setSelectedCategory: (category?: PolicyCategory) => void;
};

function DeletePolicyCategoryModal({
    isOpen,
    onClose,
    addToast,
    refreshPolicyCategories,
    selectedCategory,
    setSelectedCategory,
}: DeletePolicyCategoryModalType) {
    const [affectedPolicies, setAffectedPolicies] = useState<ListPolicy[]>([]);

    function handleDelete() {
        deletePolicyCategory(selectedCategory?.id || '')
            .then(() => {
                addToast('Successfully deleted category');
                setSelectedCategory();
                refreshPolicyCategories();
            })
            .catch((error) => {
                addToast(error.message);
            })
            .finally(() => {
                onClose();
            });
    }

    useEffect(() => {
        if (selectedCategory?.name) {
            const query = getRequestQueryStringForSearchFilter({
                Category: selectedCategory?.name,
            });
            getPolicies(query)
                .then((policies) => {
                    setAffectedPolicies(policies);
                })
                .catch((error) => {
                    addToast(error.message);
                });
        }
    }, [selectedCategory?.name, addToast]);

    return (
        <Modal
            title="Permanently delete category?"
            isOpen={isOpen}
            variant={ModalVariant.small}
            onClose={onClose}
            data-testid="delete-category-modal"
            aria-label="Permanently delete category?"
            hasNoBodyWrapper
        >
            <ModalBoxBody>
                {affectedPolicies.length > 0 && (
                    <Flex direction={{ default: 'column' }}>
                        <FlexItem>
                            This will permanently delete and remove the
                            <b> {selectedCategory?.name} </b>
                            policy category from the following policies:
                        </FlexItem>
                        <FlexItem>
                            <Panel variant="bordered" isScrollable>
                                <PanelMain>
                                    <PanelMainBody>
                                        <List component={ListComponent.ol} type={OrderType.number}>
                                            {affectedPolicies.map(({ name }) => (
                                                <ListItem key={name}>{name}</ListItem>
                                            ))}
                                        </List>
                                    </PanelMainBody>
                                </PanelMain>
                            </Panel>
                        </FlexItem>
                    </Flex>
                )}
                {affectedPolicies.length === 0 && (
                    <div>There are no policies affected by this policy category.</div>
                )}
            </ModalBoxBody>
            <ModalBoxFooter>
                <Button key="delete" variant="danger" onClick={() => handleDelete()}>
                    Delete
                </Button>
                <Button key="cancel" variant="link" onClick={onClose}>
                    Cancel
                </Button>
            </ModalBoxFooter>
        </Modal>
    );
}

export default DeletePolicyCategoryModal;
