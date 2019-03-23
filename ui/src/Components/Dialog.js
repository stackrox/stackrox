import React from 'react';
import PropTypes from 'prop-types';
import ReactModal from 'react-modal';
import { ClipLoader as Loader } from 'react-spinners';

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
        {props.isLoading ? (
            <div className="flex">
                <Loader size={14} color="currentColor" />
                {props.loadingText && <div className="ml-4">{props.loadingText}</div>}
            </div>
        ) : (
            <>
                <div className="py-4 leading-normal">{props.text}</div>
                <div className="flex justify-center">
                    {props.onCancel && (
                        <button type="button" className="btn btn-base" onClick={props.onCancel}>
                            {props.cancelText}
                        </button>
                    )}
                    {props.onConfirm && (
                        <button
                            type="button"
                            className="btn btn-success ml-4"
                            onClick={props.onConfirm}
                        >
                            {props.confirmText}
                        </button>
                    )}
                </div>
            </>
        )}
    </ReactModal>
);

Dialog.propTypes = {
    className: PropTypes.string,
    isOpen: PropTypes.bool.isRequired,
    text: PropTypes.string.isRequired,
    onCancel: PropTypes.func,
    cancelText: PropTypes.string,
    onConfirm: PropTypes.func,
    confirmText: PropTypes.string,
    isLoading: PropTypes.bool,
    loadingText: PropTypes.string
};

Dialog.defaultProps = {
    className: '',
    onCancel: null,
    cancelText: 'Cancel',
    onConfirm: null,
    confirmText: 'Confirm',
    isLoading: false,
    loadingText: null
};

export default Dialog;
