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
                description="The page shows the collected product usage data. The current usage is computed from the last metrics received from sensors, and can be accurate to about 5 minutes. The maximum usage is aggregated hourly."
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
