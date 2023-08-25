import React, { ReactElement } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { FeedbackModal, FeedbackLocale } from '@patternfly/react-user-feedback';

import { getProductBranding } from 'constants/productBranding';
import redFeedbackImage from 'images/feedback_illo.svg';
import { selectors } from 'reducers';
import { actions } from 'reducers/feedback';

const feedbackState = createStructuredSelector({
    feedback: selectors.feedbackSelector,
});

function getFeedbackTranslations(lang, branding): FeedbackLocale {
    switch (lang) {
        case 'en-us':
        default: {
            const dialogSubhead =
                branding === 'RHACS_BRANDING'
                    ? 'Help us improve Red Hat Advanced Cluster Security for Kubernetes.'
                    : 'Help us improve StackRox.';
            const experienceSubhead =
                branding === 'RHACS_BRANDING'
                    ? 'What has your RHACS experience been like so far?'
                    : 'What has your StackRox experience been like so far?';

            return {
                getSupport: 'Get support',
                back: 'Back',
                bugReported: 'Bug Reported',
                cancel: 'Cancel',
                close: 'Close',
                describeBug:
                    'Describe the bug you encountered. For urgent issues, open a support case instead.',
                describeBugUrgentCases:
                    'Describe the bug you encountered. For urgent issues, open a support case instead.',
                describeReportBug:
                    'Describe the bug you encountered. Include where it is located and what action caused it. If this issue is urgent or blocking your workflow,',
                directInfluence:
                    'Your feedback will directly influence the future of Red Hat’s products. Opt in below to hear about future research opportunities via email.',
                email: 'Email',
                enterFeedback: 'Enter your feedback',
                feedback: 'Feedback',
                feedbackSent: 'Feedback Sent',
                helpUsImproveHCC: dialogSubhead,
                howIsConsoleExperience: experienceSubhead,
                joinMailingList: 'Join mailing list',
                informDirectionDescription:
                    'By participating in feedback sessions, usability tests, and interviews with our',
                informDirection: 'Inform the direction of Red Hat',
                learnAboutResearchOpportunities:
                    'Learn about opportunities to share your feedback with our User Research Team.',
                openSupportCase: 'Support Case',
                problemProcessingRequest:
                    'There was a problem processing the request. Try reloading the page. If the problem persists, contact',
                support: 'Red Hat support',
                reportABug: 'Report a bug',
                responseSent: 'Response sent',
                researchOpportunities: 'Yes, I would like to hear about research opportunities',
                shareFeedback: 'Share feedback',
                shareYourFeedback: 'Share your feedback with us!',
                somethingWentWrong: 'Something went wrong',
                submitFeedback: 'Submit feedback',
                teamWillReviewBug:
                    'We appreciate your feedback and our team will review your report shortly',
                tellAboutExperience: 'Tell us about your experience',
                thankYouForFeedback: 'Thank you, we appreciate your feedback.',
                thankYouForInterest:
                    'Thank you for your interest in user research. You have been added to our mailing list.',
                userResearchTeam: 'User Research Team',
                weNeverSharePersonalInformation:
                    'We never share your personal information, and you can opt out at any time.',
            };
        }
    }
}

function AcsFeedbackModal(): ReactElement | null {
    const { feedback: showFeedbackModal } = useSelector(feedbackState);
    const dispatch = useDispatch();

    const { type } = getProductBranding();

    const i18nAcsEnglish = getFeedbackTranslations('en-us', type);
    return (
        <FeedbackModal
            email="test@redhat.com"
            onShareFeedback="https://console.redhat.com/self-managed-feedback-form?source=acs"
            onOpenSupportCase={
                type === 'RHACS_BRANDING'
                    ? 'https://access.redhat.com/support/cases/#/case/new/open-case?caseCreate=true'
                    : ''
            }
            isOpen={showFeedbackModal}
            feedbackImg={redFeedbackImage}
            feedbackLocale={i18nAcsEnglish}
            onClose={() => {
                dispatch(actions.setFeedbackModalVisibility(false));
            }}
        />
    );
}

export default AcsFeedbackModal;
