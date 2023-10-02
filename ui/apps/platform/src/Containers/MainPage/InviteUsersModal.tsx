import React, { ReactElement } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Modal, ModalVariant, ModalBoxBody, ModalBoxFooter, Button } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions } from 'reducers/invite';

const feedbackState = createStructuredSelector({
    invite: selectors.inviteSelector,
});

function InviteUsersModal(): ReactElement | null {
    const { invite: showInviteModal } = useSelector(feedbackState);
    const dispatch = useDispatch();

    function onClose() {
        dispatch(actions.setInviteModalVisibility(false));
    }

    return (
        <Modal
            title="Invite users"
            isOpen={showInviteModal}
            variant={ModalVariant.small}
            onClose={onClose}
            aria-label="Permanently delete category?"
            hasNoBodyWrapper
        >
            <ModalBoxBody>Invite users form</ModalBoxBody>
            <ModalBoxFooter>
                <Button key="invite" variant="primary" onClick={() => {}}>
                    Invite users
                </Button>
                <Button key="cancel" variant="link" onClick={onClose}>
                    Cancel
                </Button>
            </ModalBoxFooter>
        </Modal>
    );
}

export default InviteUsersModal;
