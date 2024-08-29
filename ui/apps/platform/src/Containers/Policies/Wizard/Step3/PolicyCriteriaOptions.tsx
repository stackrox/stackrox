import React, { useState, useEffect } from 'react';
import {
    Button,
    Divider,
    Flex,
    FlexItem,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    TreeView,
    TreeViewDataItem,
    TreeViewSearch,
} from '@patternfly/react-core';
import { PlusCircleIcon } from '@patternfly/react-icons';
import { kebabCase } from 'lodash';
import { useFormikContext } from 'formik';

import { Policy } from 'types/policy.proto';
import { Descriptor } from './policyCriteriaDescriptors';
import { getEmptyPolicyFieldCard } from '../../policies.utils';

import './PolicyCriteriaOptions.css';

function getKeysByCategory(keys) {
    const categories = {};
    keys.forEach((key) => {
        const { category } = key;
        if (categories[category]) {
            categories[category].push(key);
        } else {
            categories[category] = [key];
        }
    });
    return categories;
}

function getPolicyFieldsAsTree(existingGroups, descriptors): TreeViewDataItem[] {
    const categories = getKeysByCategory(descriptors);

    const treeList = Object.keys(categories).map((category) => ({
        name: '',
        title: category,
        id: kebabCase(category),
        children: categories[category]
            .filter((child) => {
                const alreadyUsed = existingGroups.find(
                    (group) =>
                        group.fieldName.toLowerCase() === child?.name?.toLowerCase() ||
                        group.fieldName.toLowerCase() === child?.label?.toLowerCase()
                );
                return !alreadyUsed;
            })
            .map<TreeViewDataItem>((child: Descriptor) => ({
                name: child.longName,
                title: child.shortName || child.name,
                id: kebabCase(child.shortName),
                icon: <PlusCircleIcon />,
            })),
    }));

    return treeList;
}

type PolicyCriteriaOptionsProps = {
    descriptors: Descriptor[];
    selectedSectionIndex: number;
};

function PolicyCriteriaOptions({ descriptors, selectedSectionIndex }: PolicyCriteriaOptionsProps) {
    // const [activeItems, setActiveItems] = useState<TreeViewDataItem[]>([]);
    const [allExpanded, setAllExpanded] = useState(false);
    const [filteredItems, setFilteredItems] = useState<TreeViewDataItem[]>([]);
    const [isFiltered, setIsFiltered] = useState(false);
    const { values, setFieldValue } = useFormikContext<Policy>();
    const { policyGroups } = values.policySections[selectedSectionIndex] ?? [];

    const selectedSection = values.policySections[selectedSectionIndex];

    const treeDataItems = getPolicyFieldsAsTree([], descriptors);

    useEffect(() => {
        setFilteredItems(treeDataItems);
        setIsFiltered(false);
    }, [treeDataItems]);

    function onSearch(evt) {
        const input: string = evt.target.value;
        if (input === '') {
            setFilteredItems(treeDataItems);
            setIsFiltered(false);
        } else {
            const filtered = treeDataItems.map((item) => {
                const filteredItem = { ...item };
                if (item.children && item.children.length > 0) {
                    const filteredChildren = item.children.filter((child) => {
                        const name = typeof child.name === 'string' ? child.name : '';
                        const title = typeof child.title === 'string' ? child.title : '';
                        return (
                            name.toLowerCase().includes(input.toLowerCase()) ||
                            title.toLowerCase().includes(input.toLowerCase())
                        );
                    });

                    filteredItem.children = filteredChildren;
                }

                return filteredItem;
            });

            setFilteredItems(filtered);
            setIsFiltered(true);
        }
    }

    function onSelect(evt, treeViewItem) {
        // Ignore folders for selection
        if (treeViewItem && !treeViewItem.children) {
            // setActiveItems([treeViewItem]);
            const itemKey = treeViewItem.title;
            const itemToAdd = descriptors.find(
                (descriptor) => descriptor.shortName === itemKey || descriptor.name === itemKey
            );
            const newPolicyFieldCard = getEmptyPolicyFieldCard(itemToAdd);

            addPolicyFieldCardHandler(newPolicyFieldCard);
        }
    }

    function addPolicyFieldCardHandler(fieldCard) {
        setFieldValue(`policySections[${String(selectedSectionIndex)}].policyGroups`, [
            ...policyGroups,
            fieldCard,
        ]);
    }

    const toolbar = (
        <>
            <Toolbar className="pf-v5-u-px-sm pf-v5-py-0">
                <ToolbarContent style={{ padding: 0 }}>
                    <ToolbarItem widths={{ default: '100%' }}>
                        <TreeViewSearch
                            onSearch={onSearch}
                            id="input-search"
                            name="search-input"
                            aria-label="Filter policy criteria"
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Divider component="div" className="pf-u-mb-sm pf-u-mt-md" />
        </>
    );

    return (
        <Flex
            className="pf-v5-u-w-100"
            direction={{ default: 'column' }}
            spaceItems={{ default: 'spaceItemsNone' }}
        >
            <Title headingLevel="h3" className="pf-v5-u-px-md pf-v5-u-pt-md">
                Add policy criteria to{' '}
                {selectedSection?.sectionName || `Condition ${selectedSectionIndex + 1}`}
            </Title>
            <FlexItem>
                <Button variant="link" onClick={() => setAllExpanded(!allExpanded)}>
                    {allExpanded && 'Collapse all'}
                    {!allExpanded && 'Expand all'}
                </Button>
            </FlexItem>
            <TreeView
                // activeItems={activeItems}
                data={filteredItems}
                onSelect={onSelect}
                hasSelectableNodes
                variant="compactNoBackground"
                hasGuides
                allExpanded={allExpanded || isFiltered}
                toolbar={toolbar}
            />
            {/* <Button variant="primary" onClick={addField} isDisabled={activeItems.length === 0}>
                Add policy field
            </Button> */}
        </Flex>
    );
}

export default PolicyCriteriaOptions;
