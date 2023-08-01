import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import upperFirst from 'lodash/upperFirst';
import { Alert } from '@patternfly/react-core';

import { selectors } from 'reducers';

function closeThisWindow() {
    window.close();
}

function getMessage(response) {
    const messageClass = 'flex flex-col items-left w-full';

    if (response?.error || !response?.userAttributes || !response?.roles) {
        const body = (
            <div className={messageClass}>
                <p> {upperFirst(response?.error) || 'An unrecognized error occurred.'}</p>
                {response?.error_description && <p>{response.error_description}</p>}
            </div>
        );
        return { messageBody: body, variant: 'danger', title: 'Authentication error' };
    }

    const displayAttributes = response.userAttributes.map((curr) => {
        return (
            <li key={curr.key}>
                <span id={curr.key}>{curr.key}</span>:{' '}
                <span aria-labelledby={curr.key}>
                    {Array.isArray(curr.values) ? curr.values.join(', ') : curr.values}
                </span>
            </li>
        );
    });

    // None is likely already filtered by the backend but we keep this code to be robust.
    const displayRoles = response.roles
        .filter((role) => {
            return role.name !== `None`;
        })
        .map((role) => {
            return <li key={role.name}>{role.name}</li>;
        });

    const content = (
        <>
            <p className="pb-2 mb-2 border-b border-success-700">
                <span className="font-700" id="user-id-label">
                    User ID:
                </span>{' '}
                <span aria-labelledby="user-id-label">{response?.userID}</span>
            </p>
            <p className="pb-2 mb-2 border-b border-success-700">
                <h2 className="font-700">User Attributes:</h2>
                <ul className="list-none">{displayAttributes}</ul>
            </p>
            <h2 className="font-700">User Roles:</h2>
            <ul className="list-none">{displayRoles}</ul>
        </>
    );

    if (displayRoles.length === 0) {
        const body = (
            <div className={messageClass}>
                <p>
                    Under the current configuration, the user would not be assigned any roles and
                    therefore would be unable to log in.
                </p>
                <>{content}</>
            </div>
        );
        return { messageBody: body, messageType: 'warning', title: 'WARNING' };
    }
    const body = <div className={messageClass}>{content}</div>;
    return { messageBody: body, messageType: 'success', title: 'Authentication successful' };
}

function TestLoginResultsPage({ authProviderTestResults }) {
    if (!authProviderTestResults) {
        closeThisWindow();
    }

    const { messageBody, variant, title } = getMessage(authProviderTestResults);

    return (
        <>
            <div className="flex flex-col items-center justify-center h-full theme-light">
                <div className="flex flex-col items-center pf-u-background-color-100 w-4/5 relative">
                    <div className="p-4 w-full">
                        <Alert variant={variant} isInline title={title}>
                            {messageBody}
                        </Alert>
                    </div>
                    <div className="flex flex-col items-center border-t border-base-400 p-4 w-full">
                        <p className="mb-4">
                            You may now close this window and continue working in your original
                            window.
                        </p>
                        <button
                            type="button"
                            className="btn btn-base whitespace-nowrap h-10 ml-4"
                            onClick={closeThisWindow}
                            dataTestId="button-close-window"
                        >
                            Close Window
                        </button>
                    </div>
                </div>
            </div>
        </>
    );
}

TestLoginResultsPage.propTypes = {
    authProviderTestResults: PropTypes.shape({
        userID: PropTypes.string,
        userAttributes: PropTypes.shape({}),
        roles: PropTypes.arrayOf(PropTypes.shape({ name: PropTypes.string })),
    }),
};

TestLoginResultsPage.defaultProps = {
    authProviderTestResults: null,
};

const mapStateToProps = createStructuredSelector({
    authProviderTestResults: selectors.getLoginAuthProviderTestResults,
});

export default connect(mapStateToProps, null)(TestLoginResultsPage);
