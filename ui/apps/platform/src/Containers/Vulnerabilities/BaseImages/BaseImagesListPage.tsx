import React, { useState } from 'react';
import {
    Alert,
    AlertActionCloseButton,
    AlertGroup,
    PageSection,
    Title,
    Button,
    Flex,
    FlexItem,
    Card,
    CardBody,
    SearchInput,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

import PageTitle from 'Components/PageTitle';
import useToasts from 'hooks/patternfly/useToasts';
import type { Toast } from 'hooks/patternfly/useToasts';
import BaseImagesEmptyState from './components/BaseImagesEmptyState';
import BaseImageTable from './components/BaseImageTable';
import AddBaseImageModal from './components/AddBaseImageModal';
import { useBaseImages } from './hooks/useBaseImages';

/**
 * Base Images list page - shows all tracked base images
 */
function BaseImagesListPage() {
    const { baseImages, addBaseImage, removeBaseImage } = useBaseImages();
    const [searchValue, setSearchValue] = useState('');
    const addModalToggle = useSelectToggle();
    const { toasts, addToast, removeToast } = useToasts();

    const handleAddBaseImage = (name: string) => {
        addBaseImage(name);
        addToast(`Base image ${name} added and scanning initiated`, 'success');
    };

    const handleRemoveBaseImage = (id: string) => {
        const removedImage = baseImages.find((img) => img.id === id);
        removeBaseImage(id);
        if (removedImage) {
            addToast(`Base image ${removedImage.name} removed`, 'success');
        }
    };

    const filteredBaseImages = baseImages.filter((baseImage) =>
        baseImage.name.toLowerCase().includes(searchValue.toLowerCase())
    );

    const showEmptyState = baseImages.length === 0;

    return (
        <>
            <PageTitle title="Base Images" />
            <PageSection variant="light">
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                >
                    <FlexItem>
                        <Title headingLevel="h1">Base Images</Title>
                    </FlexItem>
                    {!showEmptyState && (
                        <FlexItem>
                            <Button variant="primary" onClick={addModalToggle.openSelect}>
                                Add base image
                            </Button>
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>

            <PageSection>
                {showEmptyState ? (
                    <Card>
                        <CardBody>
                            <BaseImagesEmptyState onAddBaseImage={addModalToggle.openSelect} />
                        </CardBody>
                    </Card>
                ) : (
                    <>
                        <Toolbar>
                            <ToolbarContent>
                                <ToolbarItem variant="search-filter">
                                    <SearchInput
                                        placeholder="Search by base image name"
                                        value={searchValue}
                                        onChange={(_event, value) => setSearchValue(value)}
                                        onClear={() => setSearchValue('')}
                                    />
                                </ToolbarItem>
                            </ToolbarContent>
                        </Toolbar>
                        <Card>
                            <BaseImageTable
                                baseImages={filteredBaseImages}
                                onRemove={handleRemoveBaseImage}
                            />
                        </Card>
                    </>
                )}
            </PageSection>

            <AddBaseImageModal
                isOpen={addModalToggle.isOpen}
                onClose={addModalToggle.closeSelect}
                onAdd={handleAddBaseImage}
            />

            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }: Toast) => (
                    <Alert
                        key={key}
                        variant={variant}
                        title={title}
                        component="p"
                        timeout
                        onTimeout={() => removeToast(key)}
                        actionClose={
                            <AlertActionCloseButton
                                title={title}
                                variantLabel={variant}
                                onClose={() => removeToast(key)}
                            />
                        }
                    >
                        {children}
                    </Alert>
                ))}
            </AlertGroup>
        </>
    );
}

export default BaseImagesListPage;
