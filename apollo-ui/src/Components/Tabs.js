import React, { Component } from 'react';

class Tabs extends Component {
    constructor(props) {
        super(props);

        this.state = {
            active: props.headers[0]
        }

        this.tabClick = this.tabClick.bind(this);
    }

    getHeaders() {
        var active = this.state.active;
        var tabClick = this.tabClick;
        return this.props.headers.map(function (header, i) {
            var tabClass = active === header ? "tab tab-active" : "tab";
            return <button className={tabClass} key={header + '-' + i} onClick={() => tabClick(header)}>{header}</button>;
        });
    }

    tabClick(header) {
        this.setState({ active: header });
    }

    renderChildren() {
        return React.Children.map(this.props.children, child => {
            return React.cloneElement(child, { active: this.state.active});
        });
    }

    render() {
        return (
            <div className="overflow-auto w-full bg-white shadow">
                <div className="tab-group flex flex-row font-bold mb-3 bg-base-200">
                    {this.getHeaders()}
                </div>
                <div className="bg-white overflow-auto px-3">{this.renderChildren()}</div>
            </div>
        );
    }
    
}

export default Tabs;
