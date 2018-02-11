import React, { Component } from 'react';
import PropTypes from 'prop-types';

class Pills extends Component {
    static propTypes = {
        data: PropTypes.arrayOf(PropTypes.object),
        onActivePillsChange: PropTypes.func
    };

    static defaultProps = {
        data: [],
        onActivePillsChange: () => {}
    };

    constructor(props) {
        super(props);

        this.state = {
            data: this.props.data,
            active: {}
        };
    }

    activatePillHandler = item => () => {
        if (item.disabled) return;
        const { active } = this.state;
        if (active[item.value] === true) {
            delete active[item.value];
        } else {
            active[item.value] = true;
        }
        this.setState({ active });
        this.props.onActivePillsChange(this.state.active);
    };

    displayData() {
        const { active } = this.state;
        return this.state.data.map(item => {
            let pillClass = active[item.value]
                ? 'text-black select-none cursor-pointer p-2 m-2 bg-blue-lightest rounded-sm whitespace-no-wrap shadow-md'
                : 'text-black select-none cursor-pointer p-2 m-2 rounded-sm whitespace-no-wrap hover:bg-blue-lightest';
            if (item.disabled)
                pillClass =
                    'text-grey select-none cursor-default p-2 m-2 rounded-sm whitespace-no-wrap';
            return (
                <div
                    className={pillClass}
                    key={`${item.value}`}
                    onClick={this.activatePillHandler(item)}
                    onKeyPress={this.activatePillHandler(item)}
                    role="presentation"
                >
                    {item.text}
                </div>
            );
        });
    }

    render() {
        return <div className="pills">{this.displayData()}</div>;
    }
}

export default Pills;
