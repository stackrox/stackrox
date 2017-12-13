import React, { Component } from 'react';

class Pills extends Component {
    constructor(props) {
        super(props);

        this.state = {
            data: this.props.data,
            active: {}
        }

        this.activatePill = this.activatePill.bind(this);
    }

    displayData() {
        var active = this.state.active;
        var activatePill = this.activatePill;
        return this.state.data.map(function (item, i) {
            var pillClass = (active[item.value]) ? 'text-black select-none cursor-pointer p-2 m-2 bg-blue-lightest rounded-sm whitespace-no-wrap shadow-md' : 'text-black select-none cursor-pointer p-2 m-2 rounded-sm whitespace-no-wrap hover:bg-blue-lightest';
            if (item.disabled) pillClass = "text-grey select-none cursor-default p-2 m-2 rounded-sm whitespace-no-wrap";
            return <div className={pillClass} key={item + '-' + i} onClick={() => activatePill(item)}>{item.text}</div>;
        });
    }

    activatePill(item) {
        if(item.disabled) return;
        var active = this.state.active;
        (active[item.value] === true) ? delete active[item.value] : active[item.value] = true;
        this.setState({ active: active });
        this.props.onActivePillsChange(this.state.active);
    }

    render() {
        return <div className={`pills`}>
            {this.displayData()}
          </div>;
    }

}

export default Pills;
