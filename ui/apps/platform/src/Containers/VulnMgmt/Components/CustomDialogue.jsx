import React from 'react';
import PropTypes from 'prop-types';
import { Button, Flex, FlexItem } from '@patternfly/react-core';
import Modal from 'Components/Modal';

const CustomDialogue = (props) => {
    return (
        <Modal
            isOpen
            onRequestClose={props.onCancel}
            // "ignore-react-onclickoutside" will prevent the workflow side panel from closing due to interactions with this element that lives outside it's scope
            className={`ignore-react-onclickoutside ${props.className}`}
        >
            <h2 className="flex flex-1 font-700 text-lg p-4">{props.title}</h2>
            {props.children}
            <Flex className="flex m-4">
                {props.onConfirm && (
                    <FlexItem>
                        <Button
                            variant="primary"
                            onClick={props.onConfirm}
                            disabled={props.confirmDisabled}
                        >
                            {props.confirmText}
                        </Button>
                    </FlexItem>
                )}
                {props.onCancel && (
                    <FlexItem>
                        <Button variant="secondary" onClick={props.onCancel}>
                            {props.cancelText}
                        </Button>
                    </FlexItem>
                )}
            </Flex>
        </Modal>
    );
};

CustomDialogue.propTypes = {
    title: PropTypes.string.isRequired,
    children: PropTypes.node,
    className: PropTypes.string,
    onCancel: PropTypes.func,
    cancelText: PropTypes.string,
    onConfirm: PropTypes.func,
    confirmText: PropTypes.string,
    confirmDisabled: PropTypes.bool,
};

CustomDialogue.defaultProps = {
    children: null,
    className: '',
    onCancel: null,
    cancelText: 'Cancel',
    onConfirm: null,
    confirmText: 'Confirm',
    confirmDisabled: false,
};

export default CustomDialogue;
