import React from 'react';
import PropTypes from 'prop-types';

import ReactModal from 'react-modal';

const Modal = props => (
    <ReactModal
        isOpen={props.isOpen}
        onRequestClose={props.onRequestClose}
        contentLabel="Modal"
        ariaHideApp={false}
        overlayClassName="ReactModal__Overlay react-modal-overlay p-4 flex"
        // eslint-disable-next-line max-len
        className="ReactModal__Content w-1/3 mx-auto my-0 flex flex-col self-center bg-primary-100 overflow-hidden max-h-full transition"
    >
        {props.children}
    </ReactModal>
);

Modal.propTypes = {
    isOpen: PropTypes.bool.isRequired,
    onRequestClose: PropTypes.func.isRequired,
    children: PropTypes.node.isRequired
};

export default Modal;
