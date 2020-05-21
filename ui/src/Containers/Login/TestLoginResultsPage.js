import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { useTheme } from 'Containers/ThemeProvider';
import Message from 'Components/Message';
import { selectors } from 'reducers';
import AppWrapper from '../AppWrapper';

function closeThisWindow() {
    window.close();
}

function getMessageBody(response) {
    const headingClass = 'font-700 mb-2';

    if (response?.userAttributes) {
        const userAttributes = Object.entries(response.userAttributes);
        const displayedAttributes = userAttributes
            .map(([key, value]) => `${key}: ${value}`)
            .join(', ');
        return (
            <div>
                <h3 className={headingClass}>Authentication successful</h3>
                <p className="pb-2 mb-2 border-b border-success-700">User ID: {response?.userID}</p>
                <p>User attributes: {displayedAttributes}</p>
            </div>
        );
    }
    return (
        <div>
            <h3 className={headingClass}>Authentication error</h3>
            <p>{response?.error || 'An unrecognized error occurred.'}</p>
        </div>
    );
}

function TestLoginResultsPage({ authProviderTestResults }) {
    const { isDarkMode } = useTheme();

    if (!authProviderTestResults) {
        closeThisWindow();
    }

    const messageType = authProviderTestResults?.userAttributes ? 'info' : 'error';
    const messageBody = getMessageBody(authProviderTestResults);

    return (
        <AppWrapper>
            <section
                className={`flex flex-col items-center justify-center h-full ${
                    isDarkMode ? 'bg-base-0' : 'bg-primary-800'
                } `}
            >
                <div className="flex flex-col items-center bg-base-100 w-4/5 relative">
                    <div className="p-4 w-full">
                        <Message type={messageType} message={messageBody} />
                    </div>
                    <div className="flex flex-col items-center border-t border-base-400 p-4 w-full">
                        <p className="mb-4">
                            You may now close this window and continue working in your original
                            window.
                        </p>
                        <button
                            type="button"
                            className="btn btn-base whitespace-no-wrap h-10 ml-4"
                            onClick={closeThisWindow}
                            dataTestId="button-close-window"
                        >
                            Close Window
                        </button>
                    </div>
                </div>
            </section>
        </AppWrapper>
    );
}

TestLoginResultsPage.propTypes = {
    authProviderTestResults: PropTypes.shape({
        userID: PropTypes.string,
        userAttributes: PropTypes.shape({}),
    }),
};

TestLoginResultsPage.defaultProps = {
    authProviderTestResults: null,
};

const mapStateToProps = createStructuredSelector({
    authProviderTestResults: selectors.getLoginAuthProviderTestResults,
});

export default connect(mapStateToProps, null)(TestLoginResultsPage);
