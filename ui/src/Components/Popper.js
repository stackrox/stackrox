import React, { Component } from 'react';
import PropTypes from 'prop-types';
import onClickOutside from 'react-onclickoutside';

import { Manager, Target, Popper } from 'react-popper';

class CustomPopper extends Component {
    constructor(props) {
        super(props);

        this.state = {
            isOpen: false
        };

        this.onClick = this.onClick.bind(this);
        this.handleClickOutside = this.handleClickOutside.bind(this);
    }

    onClick() {
        this.setState(prevState => ({ isOpen: !prevState.isOpen }));
    }

    handleClickOutside() {
        this.setState({ isOpen: false });
    }

    render() {
        const { isOpen } = this.state;
        const { disabled, placement, buttonClass, buttonContent, popperContent } = this.props;

        return (
            <Manager>
                <Target>
                    <button
                        type="button"
                        data-test-id="color-picker"
                        onClick={this.onClick}
                        className={`ignore-react-onclickoutside ${buttonClass} ${
                            disabled ? 'pointer-events-none' : ''
                        }`}
                    >
                        {buttonContent}
                    </button>
                </Target>
                <Popper className={`popper z-60 ${isOpen ? '' : 'hidden'}`} placement={placement}>
                    {popperContent}
                </Popper>
            </Manager>
        );
    }
}

CustomPopper.propTypes = {
    disabled: PropTypes.bool,
    placement: PropTypes.string,
    buttonClass: PropTypes.string,
    buttonContent: PropTypes.oneOfType([PropTypes.string, PropTypes.element]).isRequired,
    popperContent: PropTypes.element.isRequired
};

CustomPopper.defaultProps = {
    disabled: false,
    placement: 'right',
    buttonClass: ''
};

const clickOutsideConfig = {
    handleClickOutside: CustomPopper.handleClickOutside
};

export default onClickOutside(CustomPopper, clickOutsideConfig);
