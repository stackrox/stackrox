import React, { Component } from 'react';
import emitter from 'emitter';
import flatten from 'flat';

class MainSidePanel extends Component {
    constructor(props) {
        super(props);

        this.state = {
            showPanel: false,
            data: {}
        }
    }

    componentDidMount() {
        // set up event listeners for this componenet
        this.tableRowSelectedListener = emitter.addListener('Table:row-selected', (data) => {
            this.setState({ showPanel: data != null, data: data });
        });
    }

    displayData() {
        if(!this.state.data) return "";
        var data = flatten(this.state.data);
        console.log(data);
        var result = Object.keys(data).map(function (key, i) {
            if (data[key] === null || data[key] === "") return "";
            return <li key={key + '-' + i}>{key}: {String(data[key])}</li>;
        });
        return result;
    }

    render() {
        return (
            <aside className={"w-1/4 h-full bg-white border border-grey-light " + ((this.state.showPanel) ? 'flex' : 'hidden')}>
                <ul>{this.displayData()}</ul>
            </aside>
        );
    }

    componentWillUnmount() {
        // remove event listeners
        this.tableRowSelectedListener.remove();
    }

}

export default MainSidePanel;
