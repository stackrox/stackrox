import React, { Component } from 'react';
import PropTypes from 'prop-types';
import onClickOutside from 'react-onclickoutside';
import { ChromePicker } from 'react-color';

import { Manager, Target, Popper } from 'react-popper';

class ColorPickerComponent extends Component {
    static propTypes = {
        id: PropTypes.string.isRequired,
        label: PropTypes.string.isRequired,
        color: PropTypes.string,
        onChange: PropTypes.func,
        disabled: PropTypes.bool,
    };

    static defaultProps = {
        color: null,
        onChange: () => {},
        disabled: false,
    };

    constructor(props) {
        super(props);
        this.state = {
            isOpen: false,
        };
    }

    handleOnChange = ({ hex }) => {
        this.props.onChange(hex, this.props.id);
    };

    handleClickOutside = () => {
        this.setState({ isOpen: false });
    };

    onClickHandler = () => {
        const { isOpen } = this.state;
        this.setState({ isOpen: !isOpen });
    };

    render() {
        return (
            <Manager>
                <Target>
                    <button
                        type="button"
                        id={this.props.id}
                        aria-label={this.props.label}
                        onClick={this.onClickHandler}
                        className={`p-1 h-5 w-full border border-base-300 ignore-react-onclickoutside ${
                            this.props.disabled ? 'pointer-events-none' : ''
                        }`}
                    >
                        <div
                            style={{ backgroundColor: this.props.color }}
                            className="h-full w-full"
                        />
                    </button>
                </Target>
                <Popper className={`popper z-10 ${this.state.isOpen ? '' : 'hidden'}`}>
                    {this.state.isOpen && (
                        <ChromePicker color={this.props.color} onChange={this.handleOnChange} />
                    )}
                </Popper>
            </Manager>
        );
    }
}

export default onClickOutside(ColorPickerComponent);
