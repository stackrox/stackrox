import React from 'react';
import PropTypes from 'prop-types';
import { ClipLoader as Loader } from 'react-spinners';
import * as Icon from 'react-feather';
import Modal from 'Components/Modal';

const CustomDialogue = (props) => {
    let confirmButtonStyle = '';
    switch (props.confirmStyle) {
        case 'alert': {
            confirmButtonStyle = 'btn-alert';
            break;
        }
        case 'success':
        default: {
            confirmButtonStyle = 'btn-success';
        }
    }

    return (
        <Modal
            isOpen
            onRequestClose={props.onCancel}
            // "ignore-react-onclickoutside" will prevent the workflow side panel from closing due to interactions with this element that lives outside it's scope
            className={`ignore-react-onclickoutside ${props.className}`}
        >
            {props.title && (
                <div className="flex items-center w-full p-3 bg-primary-700 text-xl text-base-100 uppercase">
                    <div className="flex flex-1">{props.title}</div>
                    <Icon.X className="ml-6 h-4 w-4 cursor-pointer" onClick={props.onCancel} />
                </div>
            )}
            {props.isLoading ? (
                <div className="flex p-4">
                    <Loader size={14} color="currentColor" />
                    {props.loadingText && <div className="ml-4">{props.loadingText}</div>}
                </div>
            ) : (
                <>
                    {props.children}
                    {props.text && <div className="p-4 leading-normal">{props.text}</div>}
                    <div className="flex m-4 justify-end">
                        {props.onCancel && (
                            <button
                                type="button"
                                className="btn btn-base"
                                onClick={props.onCancel}
                                data-testid="custom-modal-cancel"
                            >
                                {props.cancelText}
                            </button>
                        )}
                        {props.onConfirm && (
                            <button
                                type="button"
                                className={`btn ${confirmButtonStyle} ml-4`}
                                onClick={props.onConfirm}
                                disabled={props.confirmDisabled}
                                data-testid="custom-modal-confirm"
                            >
                                {props.confirmText}
                            </button>
                        )}
                    </div>
                </>
            )}
        </Modal>
    );
};

CustomDialogue.propTypes = {
    title: PropTypes.string.isRequired,
    children: PropTypes.node,
    className: PropTypes.string,
    text: PropTypes.string,
    onCancel: PropTypes.func,
    cancelText: PropTypes.string,
    onConfirm: PropTypes.func,
    confirmText: PropTypes.string,
    confirmDisabled: PropTypes.bool,
    confirmStyle: PropTypes.string,
    isLoading: PropTypes.bool,
    loadingText: PropTypes.string,
};

CustomDialogue.defaultProps = {
    children: null,
    className: '',
    onCancel: null,
    cancelText: 'Cancel',
    onConfirm: null,
    confirmText: 'Confirm',
    confirmDisabled: false,
    confirmStyle: 'success',
    isLoading: false,
    loadingText: null,
    text: null,
};

export default CustomDialogue;
