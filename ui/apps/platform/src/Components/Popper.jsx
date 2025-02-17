import React, { Component } from 'react';
import PropTypes from 'prop-types';
import onClickOutside from 'react-onclickoutside';

import { Manager, Target, Popper } from 'react-popper';

class CustomPopper extends Component {
    static propTypes = {
        disabled: PropTypes.bool,
        placement: PropTypes.string,
        reactOutsideClassName: PropTypes.string,
        buttonClass: PropTypes.string,
        buttonContent: PropTypes.oneOfType([PropTypes.string, PropTypes.node]).isRequired,
        popperContent: PropTypes.element.isRequired,
    };

    static defaultProps = {
        disabled: false,
        reactOutsideClassName: 'ignore-react-onclickoutside',
        placement: 'right',
        buttonClass: '',
    };

    constructor(props) {
        super(props);

        this.state = {
            isOpen: false,
        };

        this.onClick = this.onClick.bind(this);
    }

    handleClickOutside() {
        this.setState({ isOpen: false });
    }

    onClick() {
        this.setState((prevState) => ({ isOpen: !prevState.isOpen }));
    }

    render() {
        const { isOpen } = this.state;
        const {
            disabled,
            placement,
            buttonClass,
            buttonContent,
            popperContent,
            reactOutsideClassName,
        } = this.props;

        return (
            <Manager>
                <Target>
                    <button
                        type="button"
                        data-testid="popper-button"
                        onClick={this.onClick}
                        className={`${reactOutsideClassName} ${buttonClass} ${
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

const CustomPopperContainer = (props) => {
    const EnhancedCustomPopper = onClickOutside(CustomPopper);
    return (
        <EnhancedCustomPopper outsideClickIgnoreClass={props.reactOutsideClassName} {...props} />
    );
};

export default CustomPopperContainer;
