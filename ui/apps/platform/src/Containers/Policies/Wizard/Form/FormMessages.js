import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import Message from 'Components/Message';

function FormMessages({ messages }) {
    return messages.length > 0 ? (
        <div className="p-3">
            {messages.map((msg) => (
                <div key={msg.content} className="mb-2 last:mb-0">
                    <Message type={msg.type} message={msg.message} />
                </div>
            ))}
        </div>
    ) : null;
}

FormMessages.propTypes = {
    messages: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
};

const mapStateToProps = createStructuredSelector({
    messages: selectors.getFormMessages,
});

export default connect(mapStateToProps)(FormMessages);
