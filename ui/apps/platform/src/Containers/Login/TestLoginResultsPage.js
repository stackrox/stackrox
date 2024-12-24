import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import upperFirst from 'lodash/upperFirst';
import {
    Alert,
    Button,
    CodeBlock,
    CodeBlockCode,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    List,
    ListItem,
} from '@patternfly/react-core';

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
            <DescriptionListGroup key={curr.key}>
                <DescriptionListTerm>{curr.key}</DescriptionListTerm>
                <DescriptionListDescription>
                    {Array.isArray(curr.values) ? curr.values.join(', ') : curr.values}
                </DescriptionListDescription>
            </DescriptionListGroup>
        );
    });

    // None is likely already filtered by the backend but we keep this code to be robust.
    const filteredRoles = response.roles.filter((role) => role.name !== 'None');

    const content = (
        <DescriptionList isCompact isHorizontal className="pf-v5-u-pt-md">
            <DescriptionListGroup>
                <DescriptionListTerm>User ID</DescriptionListTerm>
                <DescriptionListDescription>{response?.userID}</DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>User attributes</DescriptionListTerm>
                <DescriptionListDescription>
                    <DescriptionList isCompact isHorizontal>
                        {displayAttributes}
                    </DescriptionList>
                </DescriptionListDescription>
            </DescriptionListGroup>
            {response?.idpToken && (
                <DescriptionListGroup>
                    <DescriptionListTerm>IdP token</DescriptionListTerm>
                    <DescriptionListDescription>
                        <CodeBlock>
                            <CodeBlockCode>
                                {JSON.stringify(JSON.parse(response?.idpToken), null, 2)}
                            </CodeBlockCode>
                        </CodeBlock>
                    </DescriptionListDescription>
                </DescriptionListGroup>
            )}
            {filteredRoles.length !== 0 && (
                <DescriptionListGroup>
                    <DescriptionListTerm>User roles</DescriptionListTerm>
                    <DescriptionListDescription>
                        <List isPlain>
                            {filteredRoles.map(({ name }) => (
                                <ListItem key={name}>{name}</ListItem>
                            ))}
                        </List>
                    </DescriptionListDescription>
                </DescriptionListGroup>
            )}
        </DescriptionList>
    );

    if (filteredRoles.length === 0) {
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
                <div className="flex flex-col items-center pf-v5-u-background-color-100 w-4/5 relative">
                    <div className="p-4 w-full">
                        <Alert variant={variant} isInline title={title} component="p">
                            {messageBody}
                        </Alert>
                    </div>
                    <div className="flex flex-col items-center p-4 w-full">
                        <p className="mb-4">
                            You may now close this window and continue working in your original
                            window.
                        </p>
                        <Button variant="primary" type="button" onClick={closeThisWindow}>
                            Close window
                        </Button>
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
