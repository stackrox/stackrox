import React, { ReactElement } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { FeedbackModal } from '@patternfly/react-user-feedback';

import redFeedbackImage from 'images/feedback_illo.svg';
import { selectors } from 'reducers';
import { actions } from 'reducers/feedback';

const feedbackState = createStructuredSelector({
    feedback: selectors.feedbackSelector,
});

function AcsFeedbackModal(): ReactElement | null {
    const { feedback: showFeedbackModal } = useSelector(feedbackState);
    const dispatch = useDispatch();

    return (
        <FeedbackModal
            email="test@redhat.com"
            onShareFeedback="https://console.redhat.com/self-managed-feedback-form"
            isOpen={showFeedbackModal}
            feedbackImg={redFeedbackImage}
            onClose={() => {
                dispatch(actions.setFeedbackModalVisibility(false));
            }}
        />
    );
}

export default AcsFeedbackModal;
