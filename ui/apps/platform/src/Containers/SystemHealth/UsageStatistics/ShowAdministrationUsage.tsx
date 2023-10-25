import React, { ReactElement } from 'react';
import { Button, ButtonVariant, Modal, ModalBoxBody, ModalVariant } from '@patternfly/react-core';
import useModal from 'hooks/useModal';
import AdministrationUsageForm from './AdministrationUsageForm';

function ShowAdministrationUsage(): ReactElement {
    const { isModalOpen, openModal, closeModal } = useModal();

    return (
        <>
            <Button
                key="open-select-modal"
                data-testid="administration-usage-modal-open-button"
                variant={ButtonVariant.secondary}
                onClick={openModal}
            >
                Show administration usage
            </Button>
            <Modal
                title="Administration usage"
                description="The page shows the collected administration usage data: number of secured Kubernetes nodes and CPU units. The current usage is computed from the latest metrics received from sensors, and there can be a delay of about 5 minutes. The maximum usage is aggregated hourly and only includes clusters that are still connected. The date range is inclusive and depends on the user's timezone. Data shown is not sent to Red Hat or displayed as Prometheus metrics."
                isOpen={isModalOpen}
                variant={ModalVariant.large}
                onClose={closeModal}
                aria-label="Administration usage"
                hasNoBodyWrapper
            >
                <ModalBoxBody>
                    <AdministrationUsageForm />
                </ModalBoxBody>
            </Modal>
        </>
    );
}

export default ShowAdministrationUsage;
