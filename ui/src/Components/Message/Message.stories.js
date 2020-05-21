import React from 'react';

import Message from './Message';

export default {
    title: 'Message',
    component: Message,
};

export const withStringInfoStyle = () => <Message type="info" message="This is an info message." />;

export const withStringWarnStyle = () => (
    <Message type="warn" message="This is a warning message." />
);

export const withStringErrorStyle = () => (
    <Message type="warn" message="This is an error message." />
);

export const withStringGuidanceStyle = () => (
    <Message type="guidance" message="This is a guidance message." />
);

export const withStringLoadingStyle = () => (
    <Message type="loading" message="This is a loading message." />
);

export const withElementsErrorStyle = () => {
    const message = (
        <ul className="list-decimal">
            <li>One type of error</li>
            <li>Another type of error</li>
        </ul>
    );
    return <Message type="error" message={message} />;
};
