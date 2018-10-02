import React from 'react';
import PropTypes from 'prop-types';
import ReactModal from 'react-modal';

const Modal = props => (
    <ReactModal
        isOpen={props.isOpen}
        onRequestClose={props.onRequestClose}
        contentLabel="Modal"
        ariaHideApp={false}
        overlayClassName="ReactModal__Overlay react-modal-overlay p-4 flex shadow-lg rounded-sm"
        className={`ReactModal__Content mx-auto my-0 flex flex-col self-center bg-base-100 overflow-hidden max-h-full transition ${
            props.className
        }`}
    >
        {props.children}
    </ReactModal>
);

Modal.propTypes = {
    isOpen: PropTypes.bool.isRequired,
    onRequestClose: PropTypes.func.isRequired,
    children: PropTypes.node.isRequired,
    className: PropTypes.string
};

Modal.defaultProps = {
    className: ''
};

export default Modal;
