import React, { Component } from 'react';
import * as Icon from "react-feather";


class Select extends Component {

    render() {
        return <div className="relative ml-3">
            <select className="block w-full border bg-base-100 border-base-200 text-base-500 p-3 pr-8 rounded">
              {this.props.options.map(function(option, i) {
                return <option key={option}>{option}</option>;
              })}
            </select>
            <div className="absolute pin-y pin-r flex items-center px-2">
              <Icon.ChevronDown className="h-4 w-4" />
            </div>
          </div>;
    }

}

export default Select;
