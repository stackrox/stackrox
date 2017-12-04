import React, { Component } from 'react';

class Select extends Component {
    constructor(props) {
        super(props);

        this.state = {
            options: this.props.options,
            active: {}
        }
    }

    displayOptions() {
        return this.state.options.map(function(option, i) {
            return <option key={option}>{option}</option>;
        });
    }

    render() {
        return (
            <div className="relative">
                <select className="block appearance-none w-full bg-grey-lighter border border-gray-light text-grey-darker py-2 px-4 pr-8 rounded">
                    {this.displayOptions()}
                </select>
                <div className="pointer-events-none absolute pin-y pin-r flex items-center px-2 border-grey-lighter">
                    <svg className="h-4 w-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"><path d="M9.293 12.95l.707.707L15.657 8l-1.414-1.414L10 10.828 5.757 6.586 4.343 8z" /></svg>
                </div>
            </div>
        );
    }

}

export default Select;
