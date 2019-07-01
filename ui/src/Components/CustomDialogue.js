import React from 'react';
import PropTypes from 'prop-types';
import { ClipLoader as Loader } from 'react-spinners';
import * as Icon from 'react-feather';
import Modal from 'Components/Modal';

const CustomDialogue = props => (
    <Modal isOpen onRequestClose={props.onCancel} className={props.className}>
        {props.title && (
            <div className="flex items-center w-full p-3 bg-primary-700 text-xl uppercase text-base-100 uppercase">
                <div className="flex flex-1">{props.title}</div>
                <Icon.X className="ml-6 h-4 w-4 cursor-pointer" onClick={props.onCancel} />
            </div>
        )}
        {props.isLoading ? (
            <div className="flex">
                <Loader size={14} color="currentColor" />
                {props.loadingText && <div className="ml-4">{props.loadingText}</div>}
            </div>
        ) : (
            <>
                {props.children}
                {props.text && <div className="p-4 leading-normal">{props.text}</div>}
                <div className="flex m-4 justify-end">
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
                            disabled={props.confirmDisabled}
                        >
                            {props.confirmText}
                        </button>
                    )}
                </div>
            </>
        )}
    </Modal>
);

CustomDialogue.propTypes = {
    title: PropTypes.string.isRequired,
    children: PropTypes.oneOfType([PropTypes.arrayOf(PropTypes.node), PropTypes.node]),
    className: PropTypes.string,
    text: PropTypes.string.isRequired,
    onCancel: PropTypes.func,
    cancelText: PropTypes.string,
    onConfirm: PropTypes.func,
    confirmText: PropTypes.string,
    confirmDisabled: PropTypes.bool,
    isLoading: PropTypes.bool,
    loadingText: PropTypes.string
};

CustomDialogue.defaultProps = {
    children: null,
    className: '',
    onCancel: null,
    cancelText: 'Cancel',
    onConfirm: null,
    confirmText: 'Confirm',
    confirmDisabled: false,
    isLoading: false,
    loadingText: 'null'
};

export default CustomDialogue;
