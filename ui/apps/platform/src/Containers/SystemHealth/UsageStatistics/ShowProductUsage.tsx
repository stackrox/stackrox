import React, { ReactElement } from 'react';
import { Button, ButtonVariant, Modal, ModalBoxBody, ModalVariant } from '@patternfly/react-core';
import useModal from 'hooks/useModal';
import ProductUsageForm from './ProductUsageForm';

function ShowProductUsage(): ReactElement {
    const { isModalOpen, openModal, closeModal } = useModal();

    return (
        <>
            <Button
                key="open-select-modal"
                data-testid="product-usage-modal-open-button"
                variant={ButtonVariant.secondary}
                onClick={openModal}
            >
                Show product usage
            </Button>
            <Modal
                title="Product usage"
                description="The page shows the collected product usage data: number of secured Kubernetes nodes and CPU units. The current usage is computed from the last metrics received from sensors, and can be accurate to about 5 minutes. The maximum usage is aggregated hourly and only includes clusters which are still connected. The date range is inclusive and depends on the user's timezone. Data shown is not sent anywhere to Red Hat or metrics collection."
                isOpen={isModalOpen}
                variant={ModalVariant.large}
                onClose={closeModal}
                aria-label="Product usage"
                hasNoBodyWrapper
            >
                <ModalBoxBody>
                    <ProductUsageForm />
                </ModalBoxBody>
            </Modal>
        </>
    );
}

export default ShowProductUsage;
