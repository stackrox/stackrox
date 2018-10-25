import React from 'react';
import PropTypes from 'prop-types';
import ReactModal from 'react-modal';

const Dialog = props => (
    <ReactModal
        isOpen={props.isOpen}
        contentLabel="Modal"
        ariaHideApp={false}
        overlayClassName="ReactModal__Overlay react-modal-overlay p-4 flex"
        className={`ReactModal__Content dialog mx-auto my-0 flex flex-col self-center bg-primary-100 overflow-hidden max-h-full transition p-4 ${
            props.className
        }`}
    >
        <div className="py-4 leading-normal">{props.text}</div>
        <div className="flex justify-end">
            <button type="button" className="btn btn-base" onClick={props.onCancel}>
                Cancel
            </button>
            <button type="button" className="btn btn-success" onClick={props.onConfirm}>
                Confirm
            </button>
        </div>
    </ReactModal>
);

Dialog.propTypes = {
    className: PropTypes.string,
    isOpen: PropTypes.bool.isRequired,
    text: PropTypes.string.isRequired,
    onCancel: PropTypes.func.isRequired,
    onConfirm: PropTypes.func.isRequired
};

Dialog.defaultProps = {
    className: ''
};

export default Dialog;
