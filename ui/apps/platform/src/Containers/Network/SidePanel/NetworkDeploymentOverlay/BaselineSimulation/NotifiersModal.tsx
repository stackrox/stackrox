import React, { ReactElement, useEffect, useState } from 'react';
import { useSelector } from 'react-redux';
import { createSelector } from 'reselect';
import { Alert, Button, Checkbox, Flex, Modal } from '@patternfly/react-core';

import { selectors } from 'reducers';

type Notifier = {
    id: string;
    name: string;
};

const notifiersSelector = createSelector(
    selectors.getNotifiers,
    (notifiers: Notifier[]) => notifiers || []
);

export type NotifiersModalProps = {
    closeModal: () => void;
    isModalOpen: boolean;
    sharePolicy: (selectedNotifiers: string[]) => Promise<never | Record<string, unknown>>;
};

function NotifiersModal({
    closeModal,
    isModalOpen,
    sharePolicy,
}: NotifiersModalProps): ReactElement {
    const [callState, setCallState] = useState<'network' | 'success' | 'error' | null>(null);
    const [errorMessage, setErrorMessage] = useState('');
    const availableNotifiers = useSelector(notifiersSelector);
    const [selectedNotifiers, setSelectedNotifiers] = useState<string[]>([]);

    useEffect(() => {
        setSelectedNotifiers([]);
    }, [availableNotifiers]);

    function toggleNotifier(checked, event) {
        const { target } = event;
        const value = target.checked;
        const { name } = target;

        if (value) {
            setSelectedNotifiers([...selectedNotifiers, name]);
        } else {
            setSelectedNotifiers(selectedNotifiers.filter((selected) => selected !== name));
        }
    }

    function handleShare() {
        setErrorMessage('');
        setCallState('network');
        sharePolicy(selectedNotifiers)
            .then(() => {
                setCallState('success');
                setTimeout(() => {
                    closeModal();
                }, 4000);
            })
            .catch((err) => {
                setCallState('error');
                setErrorMessage(err);
            });
    }

    const primaryDisabled = callState === 'success' || callState === 'network';
    const secondaryDisabled = callState === 'success' || callState === 'network';

    return (
        <Modal
            title="Share Network Policy YAML With Team"
            isOpen={isModalOpen}
            onClose={closeModal}
            actions={[
                <Button
                    key="confirm"
                    variant="primary"
                    onClick={handleShare}
                    isDisabled={primaryDisabled}
                >
                    Notify
                </Button>,
                <Button
                    key="cancel"
                    variant="link"
                    onClick={closeModal}
                    isDisabled={secondaryDisabled}
                >
                    Cancel
                </Button>,
            ]}
            variant="medium"
        >
            {callState === 'success' && (
                <Alert
                    variant="success"
                    timeout={3500}
                    isLiveRegion
                    title="YAML has been shared."
                />
            )}
            {callState === 'error' && (
                <Alert variant="danger" isLiveRegion title={`An error occurred. ${errorMessage}`} />
            )}
            <Flex className="pf-u-p-lg">
                {availableNotifiers.map((notifier) => {
                    const isChecked = !!selectedNotifiers.find(
                        (selected) => selected === notifier.id
                    );
                    return (
                        <Checkbox
                            isChecked={isChecked}
                            onChange={toggleNotifier}
                            label={notifier.name}
                            id={notifier.id}
                            name={notifier.id}
                        />
                    );
                })}
            </Flex>
        </Modal>
    );
}

export default NotifiersModal;
