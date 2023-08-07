import React, { ReactElement, useState } from 'react';
import { Button, ButtonVariant, Modal, ModalBoxBody, ModalVariant } from '@patternfly/react-core';
import UsageStatisticsForm from './UsageStatisticsForm';

function ShowProductUsage(): ReactElement {
    const [isModalOpen, setIsModalOpen] = useState(false);

    function onCloseModalHandler() {
        setIsModalOpen(false);
    }

    return (
        <>
            <Button
                key="open-select-modal"
                data-testid="product-usage-modal-open-button"
                variant={ButtonVariant.secondary}
                onClick={() => {
                    setIsModalOpen(true);
                }}
            >
                Show product usage
            </Button>
            <Modal
                title="Product usage"
                description="Lorem ipsum dolor sit amet, consectetur adipiscing elit. Proin blandit augue est. Duis fermentum lacinia consequat. Donec id felis elit. Donec nec nunc felis. Morbi quis enim scelerisque, ullamcorper velit."
                isOpen={isModalOpen}
                variant={ModalVariant.large}
                onClose={onCloseModalHandler}
                aria-label="Product usage"
                hasNoBodyWrapper
            >
                <ModalBoxBody>
                    <UsageStatisticsForm />
                </ModalBoxBody>
            </Modal>
        </>
    );
}

export default ShowProductUsage;
